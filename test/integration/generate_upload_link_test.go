package integration

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/fhuszti/medias-ms-go/internal/handler/api"
	"github.com/fhuszti/medias-ms-go/internal/migration"
	"github.com/fhuszti/medias-ms-go/internal/model"
	"github.com/fhuszti/medias-ms-go/internal/port"
	"github.com/fhuszti/medias-ms-go/internal/repository/mariadb"
	mediaService "github.com/fhuszti/medias-ms-go/internal/usecase/media"
	msuuid "github.com/fhuszti/medias-ms-go/internal/uuid"
	"github.com/fhuszti/medias-ms-go/test/testutil"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

func TestGenerateUploadLinkIntegration_Success(t *testing.T) {
	testDB, err := testutil.SetupTestDB()
	if err != nil {
		t.Fatalf("setup DB: %v", err)
	}
	defer testDB.Cleanup()
	database := testDB.DB
	if err := migration.MigrateUp(database); err != nil {
		t.Fatalf("could not run migrations: %v", err)
	}

	bCleanup, err := testutil.SetupTestBuckets(GlobalStrg)
	if err != nil {
		t.Fatalf("setup buckets: %v", err)
	}
	defer bCleanup()

	mediaRepo := mariadb.NewMediaRepository(database)
	svc := mediaService.NewUploadLinkGenerator(mediaRepo, GlobalStrg, msuuid.NewUUID)

	in := port.GenerateUploadLinkInput{
		Name: "file_example.png",
	}

	out, err := svc.GenerateUploadLink(context.Background(), in)
	if err != nil {
		t.Fatalf("GenerateUploadLink returned error: %v", err)
	}

	if out.ID == msuuid.UUID(uuid.Nil) {
		t.Fatal("expected non-empty ID")
	}

	if out.URL == "" {
		t.Fatal("expected non-empty presigned URL")
	}
	u, err := url.Parse(out.URL)
	if err != nil {
		t.Fatalf("invalid URL %q: %v", out.URL, err)
	}
	parts := strings.Split(strings.TrimPrefix(u.Path, "/"), "/")
	if len(parts) != 2 {
		t.Fatalf("unexpected URL path: %s", u.Path)
	}
	bucketName, objectKey := parts[0], parts[1]
	if bucketName != "staging" {
		t.Errorf("expected bucket 'staging', got %q", bucketName)
	}
	if objectKey != out.ID.String() {
		t.Errorf("expected objectKey to be %q, got %q", objectKey, out.ID.String())
	}

	var (
		id               msuuid.UUID
		bucket           string
		originalFilename string
		status           model.MediaStatus
		metadata         model.Metadata
		variants         model.Variants
	)
	row := testDB.DB.QueryRowContext(context.Background(),
		"SELECT id, bucket, original_filename, status, metadata, variants FROM medias WHERE object_key = ?", objectKey)
	if err := row.Scan(&id, &bucket, &originalFilename, &status, &metadata, &variants); err != nil {
		t.Fatalf("failed to scan media record: %v", err)
	}

	if id != out.ID {
		t.Errorf("expected ID %q, got %q", out.ID, id)
	}
	if bucket != "staging" {
		t.Errorf("bucket should be 'staging', got %q", bucket)
	}
	if originalFilename != in.Name {
		t.Errorf("expected originalFilename to be %q', got %q", originalFilename, in.Name)
	}
	if status != model.MediaStatusPending {
		t.Errorf("expected status %q, got %q", model.MediaStatusPending, status)
	}
	if !reflect.DeepEqual(metadata, model.Metadata{}) {
		t.Errorf("expected empty Metadata struct, got %+v", metadata)
	}
	if !reflect.DeepEqual(variants, model.Variants{}) {
		t.Errorf("expected empty Variants slice, got %+v", variants)
	}
}

func TestGenerateUploadLinkIntegration_ErrorValidation(t *testing.T) {
	r := chi.NewRouter()
	r.Post("/medias/generate_upload_link", api.GenerateUploadLinkHandler(nil))

	// Missing `name` entirely
	req := httptest.NewRequest(http.MethodPost, "/medias/generate_upload_link", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	res := rec.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d; want %d", res.StatusCode, http.StatusBadRequest)
	}

	var errMap1 map[string]string
	if err := json.NewDecoder(res.Body).Decode(&errMap1); err != nil {
		t.Fatalf("decoding validation JSON: %v", err)
	}
	msgs1, ok := errMap1["name"]
	if !ok {
		t.Fatalf("expected a \"name\" key in error map, got %v", errMap1)
	}
	if !strings.Contains(msgs1, "required") {
		t.Errorf("Name error = %q; want to mention \"required\"", msgs1)
	}

	// Too-long name (>80 chars)
	longName := strings.Repeat("x", 81)
	body := `{"name":` + strconv.Quote(longName) + `}`
	req2 := httptest.NewRequest(http.MethodPost, "/medias/generate_upload_link", strings.NewReader(body))
	req2.Header.Set("Content-Type", "application/json")
	rec2 := httptest.NewRecorder()
	r.ServeHTTP(rec2, req2)

	res2 := rec2.Result()
	defer res2.Body.Close()

	if res2.StatusCode != http.StatusBadRequest {
		t.Fatalf("status (long name) = %d; want %d", res2.StatusCode, http.StatusBadRequest)
	}

	var errMap2 map[string]string
	if err := json.NewDecoder(res2.Body).Decode(&errMap2); err != nil {
		t.Fatalf("decoding validation JSON (long name): %v", err)
	}
	msgs2, ok := errMap2["name"]
	if !ok {
		t.Fatalf("expected a \"name\" key in error map for long name, got %v", errMap2)
	}
	if !strings.Contains(msgs2, "max") {
		t.Errorf("Name error (long) = %q; want to mention max length", msgs2)
	}
}
