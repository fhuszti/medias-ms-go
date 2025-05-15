package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/fhuszti/medias-ms-go/internal/db"
	mediaHandler "github.com/fhuszti/medias-ms-go/internal/handler/media"
	"github.com/fhuszti/medias-ms-go/internal/migration"
	"github.com/fhuszti/medias-ms-go/internal/model"
	"github.com/fhuszti/medias-ms-go/internal/repository/mariadb"
	mediaService "github.com/fhuszti/medias-ms-go/internal/usecase/media"
	"github.com/fhuszti/medias-ms-go/test/testutil"
)

func TestFinaliseUploadE2E(t *testing.T) {
	ctx := context.Background()

	testDB, err := testutil.SetupTestDB()
	if err != nil {
		t.Fatalf("setup DB: %v", err)
	}
	defer testDB.Cleanup()
	if err := migration.MigrateUp(testDB.DB); err != nil {
		t.Fatalf("run migrations: %v", err)
	}

	tb, err := testutil.SetupTestBuckets(GlobalMinioClient)
	if err != nil {
		t.Fatalf("setup buckets: %v", err)
	}
	defer func() {
		if err := tb.Cleanup(); err != nil {
			t.Fatalf("cleanup buckets: %v", err)
		}
	}()

	mediaRepo := mariadb.NewMediaRepository(testDB.DB)
	stgStrg, err := tb.StrgClient.WithBucket("staging")
	if err != nil {
		t.Fatalf("init staging bucket: %v", err)
	}
	getDest := func(b string) (mediaService.Storage, error) {
		return tb.StrgClient.WithBucket(b)
	}
	svc := mediaService.NewUploadFinaliser(mediaRepo, stgStrg, getDest)

	id := db.UUID(uuid.MustParse("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"))
	objectKey := "doc.md"
	content := []byte("# Hello E2E Test")
	prepareDataForTest(id, objectKey, content, ctx, t, mediaRepo, stgStrg)

	r := chi.NewRouter()
	r.With(mediaHandler.WithDestBucket).
		Post("/medias/{destBucket}/complete", mediaHandler.FinaliseUploadHandler(svc))
	srv := httptest.NewServer(r)
	defer srv.Close()

	url := fmt.Sprintf("%s/medias/%s/complete", srv.URL, "images")
	reqBody := fmt.Sprintf(`{"id":"%s"}`, id.String())
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBufferString(reqBody))
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("http request failed: %v", err)
	}
	defer resp.Body.Close()

	// Assert HTTP response
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d; want %d", resp.StatusCode, http.StatusOK)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q; want %q", ct, "application/json")
	}
	var got model.Media
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatalf("decode JSON body: %v", err)
	}
	if got.ID != id {
		t.Errorf("resp ID = %v; want %v", got.ID, id)
	}
	if got.Status != model.MediaStatusCompleted {
		t.Errorf("resp Status = %q; want %q", got.Status, model.MediaStatusCompleted)
	}
	if got.SizeBytes == nil || *got.SizeBytes != int64(len(content)) {
		t.Errorf("resp SizeBytes = %v; want %d", got.SizeBytes, len(content))
	}
	if got.MimeType == nil || *got.MimeType != "text/markdown" {
		t.Errorf("resp MimeType = %q; want %q", *got.MimeType, "text/markdown")
	}

	// Assert DB updated
	saved, err := mediaRepo.GetByID(ctx, id)
	if err != nil {
		t.Fatalf("GetByID after: %v", err)
	}
	if saved.Status != model.MediaStatusCompleted {
		t.Errorf("db Status = %q; want %q", saved.Status, model.MediaStatusCompleted)
	}

	// Assert file moved & content intact
	destStrg, err := getDest("images")
	if err != nil {
		t.Fatalf("init dest bucket: %v", err)
	}
	exists, err := destStrg.FileExists(ctx, objectKey)
	if err != nil {
		t.Fatalf("checking dest FileExists: %v", err)
	}
	if !exists {
		t.Error("expected file in destination bucket")
	}
	still, err := stgStrg.FileExists(ctx, objectKey)
	if err != nil {
		t.Fatalf("checking staging FileExists: %v", err)
	}
	if still {
		t.Error("expected staging file removed")
	}
	rc, err := destStrg.GetFile(ctx, objectKey)
	if err != nil {
		t.Fatalf("GetFile on dest: %v", err)
	}
	dataOut, err := io.ReadAll(rc)
	if err != nil {
		t.Fatalf("reading dest file: %v", err)
	}
	rc.Close()
	if !bytes.Equal(dataOut, content) {
		t.Errorf("dest content = %q; want %q", dataOut, content)
	}
}

func prepareDataForTest(id db.UUID, objectKey string, content []byte, ctx context.Context, t *testing.T, mediaRepo *mariadb.MediaRepository, stgStrg mediaService.Storage) {
	m := &model.Media{
		ID:        id,
		ObjectKey: objectKey,
		Status:    model.MediaStatusPending,
	}
	if err := mediaRepo.Create(ctx, m); err != nil {
		t.Fatalf("insert media: %v", err)
	}

	// upload into "staging"
	if err := stgStrg.SaveFile(
		ctx,
		objectKey,
		bytes.NewReader(content),
		int64(len(content)),
		map[string]string{
			"Content-Type": "text/markdown",
		},
	); err != nil {
		t.Fatalf("upload to staging: %v", err)
	}
}
