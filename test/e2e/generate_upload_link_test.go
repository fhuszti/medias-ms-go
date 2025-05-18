package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/fhuszti/medias-ms-go/internal/db"
	mediaHandler "github.com/fhuszti/medias-ms-go/internal/handler/media"
	"github.com/fhuszti/medias-ms-go/internal/migration"
	"github.com/fhuszti/medias-ms-go/internal/model"
	"github.com/fhuszti/medias-ms-go/internal/repository/mariadb"
	mediaService "github.com/fhuszti/medias-ms-go/internal/usecase/media"
	"github.com/fhuszti/medias-ms-go/test/testutil"
	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"testing"
)

func TestGenerateUploadLinkE2E(t *testing.T) {
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
	defer func() {
		if err := tb.Cleanup(); err != nil {
			t.Fatalf("cleanup buckets: %v", err)
		}
	}()

	mediaRepo := mariadb.NewMediaRepository(database)
	strg, err := tb.StrgClient.WithBucket("staging")
	if err != nil {
		t.Fatalf("failed to initialise bucket 'staging': %v", err)
	}
	svc := mediaService.NewUploadLinkGenerator(mediaRepo, strg)
	h := mediaHandler.GenerateUploadLinkHandler(svc)

	srv := httptest.NewServer(h)
	defer srv.Close()

	reqBody := map[string]interface{}{
		"name": "file_example.pdf",
	}
	b, err := json.Marshal(reqBody)
	if err != nil {
		t.Fatalf("could not marshal request: %v", err)
	}

	resp, err := http.Post(srv.URL, "application/json", bytes.NewReader(b))
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", resp.StatusCode)
	}

	var out mediaService.GenerateUploadLinkOutput
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("could not decode response: %v", err)
	}

	if out.ID == db.UUID(uuid.Nil) {
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

	payload := []byte("hello-minio")
	uploadToPresignedURL(t, out.URL, payload)
	obj, err := tb.StrgClient.Client.GetObject(context.Background(), bucketName, objectKey, minio.GetObjectOptions{})
	if err != nil {
		t.Fatalf("failed to get object: %v", err)
	}
	data, err := io.ReadAll(obj)
	if err != nil {
		t.Fatalf("failed to read object: %v", err)
	}
	if !bytes.Equal(data, payload) {
		t.Errorf("object content mismatch: expected %q, got %q", payload, data)
	}

	var (
		id               db.UUID
		originalFilename string
		status           model.MediaStatus
		metadata         model.Metadata
	)
	row := testDB.DB.QueryRowContext(context.Background(),
		"SELECT id, original_filename, status, metadata FROM medias WHERE object_key = ?", objectKey)
	if err := row.Scan(&id, &originalFilename, &status, &metadata); err != nil {
		t.Fatalf("failed to scan media record: %v", err)
	}

	if id != out.ID {
		t.Errorf("expected ID %q, got %q", out.ID, id)
	}
	if originalFilename != "file_example.pdf" {
		t.Errorf("expected originalFilename to be 'file_example.pdf', got %q", originalFilename)
	}
	if status != model.MediaStatusPending {
		t.Errorf("expected status %q, got %q", model.MediaStatusPending, status)
	}
	if !reflect.DeepEqual(metadata, model.Metadata{}) {
		t.Errorf("expected empty Metadata struct, got %+v", metadata)
	}
}

func uploadToPresignedURL(t *testing.T, presignedURL string, payload []byte) {
	req, err := http.NewRequest(http.MethodPut, presignedURL, bytes.NewReader(payload))
	if err != nil {
		t.Fatalf("failed to create PUT request: %v", err)
	}

	req.Header.Set("Content-Type", "text/markdown")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("failed to PUT object: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected status 200 or 204; got %d", resp.StatusCode)
	}
}
