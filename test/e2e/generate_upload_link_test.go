package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	mediaHandler "github.com/fhuszti/medias-ms-go/internal/handler/media"
	"github.com/fhuszti/medias-ms-go/internal/migration"
	"github.com/fhuszti/medias-ms-go/internal/model"
	"github.com/fhuszti/medias-ms-go/internal/repository/mariadb"
	mediaService "github.com/fhuszti/medias-ms-go/internal/usecase/media"
	"github.com/fhuszti/medias-ms-go/test/testutil"
	"github.com/minio/minio-go/v7"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
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
		"name": "file_example",
		"type": "text/markdown",
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

	var presignedURL string
	if err := json.NewDecoder(resp.Body).Decode(&presignedURL); err != nil {
		t.Fatalf("could not decode response: %v", err)
	}

	if presignedURL == "" {
		t.Fatal("expected non-empty presigned URL")
	}
	u, err := url.Parse(presignedURL)
	if err != nil {
		t.Fatalf("invalid URL %q: %v", presignedURL, err)
	}
	parts := strings.Split(strings.TrimPrefix(u.Path, "/"), "/")
	if len(parts) != 2 {
		t.Fatalf("unexpected URL path: %s", u.Path)
	}
	bucketName, objectKey := parts[0], parts[1]
	if bucketName != "staging" {
		t.Errorf("expected bucket 'staging', got %q", bucketName)
	}
	if !strings.HasPrefix(objectKey, "file_example_") {
		t.Errorf("expected objectkey to start with 'file_example_', got %q", objectKey)
	}

	payload := []byte("hello-minio")
	uploadToPresignedURL(t, presignedURL, payload)
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
		id       string
		mimeType string
		status   model.MediaStatus
	)
	row := testDB.DB.QueryRowContext(context.Background(),
		"SELECT id, mime_type, status FROM medias WHERE object_key = ?", objectKey)
	if err := row.Scan(&id, &mimeType, &status); err != nil {
		t.Fatalf("failed to scan media record: %v", err)
	}
	if mimeType != "text/markdown" {
		t.Errorf("expected mime type text/markdown, got %q", mimeType)
	}
	if status != model.MediaStatusPending {
		t.Errorf("expected status %q, got %q", model.MediaStatusPending, status)
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
