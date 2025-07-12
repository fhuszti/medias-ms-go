package integration

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/fhuszti/medias-ms-go/internal/port"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/fhuszti/medias-ms-go/internal/cache"
	"github.com/fhuszti/medias-ms-go/internal/handler/api"
	"github.com/fhuszti/medias-ms-go/internal/migration"
	"github.com/fhuszti/medias-ms-go/internal/model"
	"github.com/fhuszti/medias-ms-go/internal/repository/mariadb"
	mediaSvc "github.com/fhuszti/medias-ms-go/internal/usecase/media"
	"github.com/fhuszti/medias-ms-go/internal/uuid"
	"github.com/fhuszti/medias-ms-go/test/testutil"
	"github.com/go-chi/chi/v5"
	guuid "github.com/google/uuid"
)

func setupMediaDeleter(t *testing.T) (*mariadb.MediaRepository, port.MediaDeleter, func()) {
	t.Helper()

	testDB, err := testutil.SetupTestDB()
	if err != nil {
		t.Fatalf("setup DB: %v", err)
	}
	if err := migration.MigrateUp(testDB.DB); err != nil {
		t.Fatalf("could not run migrations: %v", err)
	}

	bCleanup, err := testutil.SetupTestBuckets(GlobalStrg)
	if err != nil {
		t.Fatalf("setup buckets: %v", err)
	}

	repo := mariadb.NewMediaRepository(testDB.DB)
	svc := mediaSvc.NewMediaDeleter(repo, cache.NewNoop(), GlobalStrg)

	cleanup := func() {
		_ = bCleanup()
		_ = testDB.Cleanup()
	}

	return repo, svc, cleanup
}

func TestDeleteMediaIntegration_Success(t *testing.T) {
	ctx := context.Background()

	repo, svc, cleanup := setupMediaDeleter(t)
	defer cleanup()

	id := uuid.UUID(guuid.MustParse("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"))
	objectKey := id.String() + ".png"
	bucket := "images"

	width, height := 32, 16
	content := testutil.GeneratePNG(t, width, height)
	size := int64(len(content))

	variants := []model.Variant{
		{Width: 16, Height: 8, ObjectKey: fmt.Sprintf("variants/%s_16.png", id)},
		{Width: 8, Height: 4, ObjectKey: fmt.Sprintf("variants/%s_8.png", id)},
	}
	for i := range variants {
		v := &variants[i]
		data := testutil.GeneratePNG(t, v.Width, v.Height)
		v.SizeBytes = int64(len(data))
		if err := GlobalStrg.SaveFile(ctx, bucket, v.ObjectKey, bytes.NewReader(data), v.SizeBytes, map[string]string{"Content-Type": "image/png"}); err != nil {
			t.Fatalf("upload variant %s: %v", v.ObjectKey, err)
		}
	}

	if err := GlobalStrg.SaveFile(ctx, bucket, objectKey, bytes.NewReader(content), size, map[string]string{"Content-Type": "image/png"}); err != nil {
		t.Fatalf("upload original: %v", err)
	}

	mime := "image/png"
	m := &model.Media{
		ID:               id,
		ObjectKey:        objectKey,
		Bucket:           bucket,
		OriginalFilename: "orig.png",
		MimeType:         &mime,
		SizeBytes:        &size,
		Status:           model.MediaStatusCompleted,
		Metadata:         model.Metadata{Width: width, Height: height},
		Variants:         variants,
	}
	if err := repo.Create(ctx, m); err != nil {
		t.Fatalf("insert media: %v", err)
	}

	if err := svc.DeleteMedia(ctx, id); err != nil {
		t.Fatalf("DeleteMedia returned error: %v", err)
	}

	if _, err := repo.GetByID(ctx, id); !errors.Is(err, sql.ErrNoRows) {
		t.Errorf("expected ErrNoRows after delete, got %v", err)
	}

	exists, err := GlobalStrg.FileExists(ctx, bucket, objectKey)
	if err != nil {
		t.Fatalf("check original exists: %v", err)
	}
	if exists {
		t.Error("original file still exists after deletion")
	}
	for _, v := range variants {
		ex, err := GlobalStrg.FileExists(ctx, bucket, v.ObjectKey)
		if err != nil {
			t.Fatalf("check variant %s: %v", v.ObjectKey, err)
		}
		if ex {
			t.Errorf("variant %s still exists", v.ObjectKey)
		}
	}
}

func TestDeleteMediaIntegration_ErrorNotFound(t *testing.T) {
	_, svc, cleanup := setupMediaDeleter(t)
	defer cleanup()

	r := chi.NewRouter()
	r.With(api.WithID()).Delete("/medias/{id}", api.DeleteMediaHandler(svc))

	id := uuid.NewUUID().String()
	req := httptest.NewRequest(http.MethodDelete, "/medias/"+id, nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	res := rec.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusNotFound {
		t.Errorf("status = %d; want %d", res.StatusCode, http.StatusNotFound)
	}

	var resp errorResponse
	if err := json.NewDecoder(res.Body).Decode(&resp); err != nil {
		t.Fatalf("decode JSON: %v", err)
	}
	if !strings.Contains(resp.Error, "Media not found") {
		t.Errorf("error = %q; want contain %q", resp.Error, "Media not found")
	}
	if cc := res.Header.Get("Cache-Control"); cc != "no-store, max-age=0, must-revalidate" {
		t.Errorf("Cache-Control = %q; want no-store...", cc)
	}
}

func TestDeleteMediaIntegration_ErrorInvalidID(t *testing.T) {
	repo := mariadb.NewMediaRepository(nil)
	svc := mediaSvc.NewMediaDeleter(repo, nil, nil)

	r := chi.NewRouter()
	r.With(api.WithID()).Delete("/medias/{id}", api.DeleteMediaHandler(svc))

	req := httptest.NewRequest(http.MethodDelete, "/medias/not-a-uuid", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	res := rec.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d; want %d", res.StatusCode, http.StatusBadRequest)
	}

	var resp errorResponse
	if err := json.NewDecoder(res.Body).Decode(&resp); err != nil {
		t.Fatalf("decode JSON: %v", err)
	}
	want := `ID "not-a-uuid" is not a valid UUID`
	if resp.Error != want {
		t.Errorf("error = %q; want %q", resp.Error, want)
	}
	if cc := res.Header.Get("Cache-Control"); cc != "no-store, max-age=0, must-revalidate" {
		t.Errorf("Cache-Control = %q; want no-store...", cc)
	}
}
