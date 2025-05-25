package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/fhuszti/medias-ms-go/internal/db"
	mediaHandler "github.com/fhuszti/medias-ms-go/internal/handler/media"
	"github.com/fhuszti/medias-ms-go/internal/migration"
	"github.com/fhuszti/medias-ms-go/internal/model"
	"github.com/fhuszti/medias-ms-go/internal/repository/mariadb"
	mediaSvc "github.com/fhuszti/medias-ms-go/internal/usecase/media"
	"github.com/fhuszti/medias-ms-go/test/testutil"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestGetMediaIntegration_SuccessDocument(t *testing.T) {
	ctx := context.Background()

	testDB, err := testutil.SetupTestDB()
	if err != nil {
		t.Fatalf("setup DB: %v", err)
	}
	defer testDB.Cleanup()
	database := testDB.DB
	if err := migration.MigrateUp(database); err != nil {
		t.Fatalf("could not run migrations: %v", err)
	}

	tb, err := testutil.SetupTestBuckets(GlobalMinioClient)
	if err != nil {
		t.Fatalf("setup buckets: %v", err)
	}
	defer tb.Cleanup()

	mediaRepo := mariadb.NewMediaRepository(database)
	getStrg := func(bucket string) (mediaSvc.Storage, error) {
		return tb.StrgClient.WithBucket(bucket)
	}
	svc := mediaSvc.NewMediaGetter(mediaRepo, getStrg)

	id := db.UUID(uuid.MustParse("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"))
	objectKey := id.String() + ".md"
	const bucket = "docs"
	strg, _ := getStrg(bucket)
	content := []byte("# Integration Test Markdown\n" + strings.Repeat(".", 512))

	m := &model.Media{
		ID:        id,
		ObjectKey: objectKey,
		Bucket:    bucket,
		Status:    model.MediaStatusCompleted,
		Metadata:  model.Metadata{},              // no width/height for docs
		SizeBytes: ptrInt64(int64(len(content))), // must match content length
		MimeType:  ptrString("text/markdown"),
	}
	if err := mediaRepo.Create(ctx, m); err != nil {
		t.Fatalf("insert media: %v", err)
	}

	if err := strg.SaveFile(ctx, objectKey, bytes.NewReader(content), int64(len(content)), map[string]string{
		"Content-Type": "text/markdown",
	}); err != nil {
		t.Fatalf("upload to %q bucket: %v", bucket, err)
	}

	out, err := svc.GetMedia(ctx, mediaSvc.GetMediaInput{ID: id})
	if err != nil {
		t.Fatalf("GetMedia returned error: %v", err)
	}

	// Assert the output
	if out.URL == "" {
		t.Errorf("expected non-empty URL, got %q", out.URL)
	}
	if !strings.Contains(out.URL, objectKey) {
		t.Errorf("URL = %q; want to contain %q", out.URL, objectKey)
	}
	// ValidUntil should be in the future but within the next 3h
	if time.Until(out.ValidUntil) <= 0 || time.Until(out.ValidUntil) > 3*time.Hour {
		t.Errorf("ValidUntil = %v; want within next 3h", out.ValidUntil)
	}
	if out.Metadata.MimeType != "text/markdown" {
		t.Errorf("MimeType = %q; want %q", out.Metadata.MimeType, "text/markdown")
	}
	if out.Metadata.SizeBytes != int64(len(content)) {
		t.Errorf("SizeBytes = %d; want %d", out.Metadata.SizeBytes, len(content))
	}
	// documents should have no variants
	if len(out.Variants) != 0 {
		t.Errorf("Variants length = %d; want 0 for documents", len(out.Variants))
	}
}

func TestGetMediaIntegration_NotFound(t *testing.T) {
	testDB, _ := testutil.SetupTestDB()
	defer testDB.Cleanup()
	migration.MigrateUp(testDB.DB)

	tb, _ := testutil.SetupTestBuckets(GlobalMinioClient)
	defer tb.Cleanup()

	repo := mariadb.NewMediaRepository(testDB.DB)
	getStrg := func(b string) (mediaSvc.Storage, error) {
		return tb.StrgClient.WithBucket(b)
	}
	svc := mediaSvc.NewMediaGetter(repo, getStrg)

	r := chi.NewRouter()
	r.With(mediaHandler.WithID()).Get("/medias/{id}", mediaHandler.GetMediaHandler(svc))
	ts := httptest.NewServer(r)
	defer ts.Close()

	// call with a UUID that does not exist in DB
	id := uuid.NewString()
	res, err := http.Get(ts.URL + "/medias/" + id)
	if err != nil {
		t.Fatalf("GET request error: %v", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusNotFound {
		t.Errorf("status = %d; want %d", res.StatusCode, http.StatusNotFound)
	}
	if ct := res.Header.Get("Cache-Control"); ct != "no-store, max-age=0, must-revalidate" {
		t.Errorf("Cache-Control = %q; want no-store...", ct)
	}
	var errResp map[string]string
	json.NewDecoder(res.Body).Decode(&errResp)
	if !strings.Contains(errResp["error"], "Media not found") {
		t.Errorf("error = %q; want contain %q", errResp["error"], "Media not found")
	}
}

func TestGetMediaIntegration_InvalidID(t *testing.T) {
	// no DB or bucket setup needed, middleware will reject
	repo := mariadb.NewMediaRepository(nil)
	svc := mediaSvc.NewMediaGetter(repo, nil)

	r := chi.NewRouter()
	r.With(mediaHandler.WithID()).Get("/medias/{id}", mediaHandler.GetMediaHandler(svc))
	ts := httptest.NewServer(r)
	defer ts.Close()

	res, err := http.Get(ts.URL + "/medias/not-a-uuid")
	if err != nil {
		t.Fatalf("GET request error: %v", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d; want %d", res.StatusCode, http.StatusBadRequest)
	}
	var errResp map[string]string
	json.NewDecoder(res.Body).Decode(&errResp)
	want := `ID "not-a-uuid" is not a valid UUID`
	if errResp["error"] != want {
		t.Errorf("error = %q; want %q", errResp["error"], want)
	}
	if cc := res.Header.Get("Cache-Control"); cc != "no-store, max-age=0, must-revalidate" {
		t.Errorf("Cache-Control = %q; want no-store...", cc)
	}
}

// helpers to get pointers
func ptrString(s string) *string { return &s }
func ptrInt64(i int64) *int64    { return &i }
