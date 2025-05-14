package integration

import (
	"context"
	"github.com/fhuszti/medias-ms-go/internal/db"
	"github.com/fhuszti/medias-ms-go/internal/migration"
	"github.com/fhuszti/medias-ms-go/internal/model"
	"github.com/fhuszti/medias-ms-go/internal/repository/mariadb"
	mediaService "github.com/fhuszti/medias-ms-go/internal/usecase/media"
	"github.com/fhuszti/medias-ms-go/test/testutil"
	"github.com/google/uuid"
	"net/url"
	"strings"
	"testing"
)

func TestGenerateUploadLinkIntegration(t *testing.T) {
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

	in := mediaService.GenerateUploadLinkInput{
		Name: "file_example",
	}

	out, err := svc.GenerateUploadLink(context.Background(), in)
	if err != nil {
		t.Fatalf("GenerateUploadLink returned error: %v", err)
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
	if !strings.HasPrefix(objectKey, "file_example_") {
		t.Errorf("expected objectkey to start with '%s_', got %q", in.Name, objectKey)
	}

	var (
		id     db.UUID
		status model.MediaStatus
	)
	row := testDB.DB.QueryRowContext(context.Background(),
		"SELECT id, status FROM medias WHERE object_key = ?", objectKey)
	if err := row.Scan(&id, &status); err != nil {
		t.Fatalf("failed to scan media record: %v", err)
	}

	if id != out.ID {
		t.Errorf("expected ID %q, got %q", out.ID, id)
	}
	if status != model.MediaStatusPending {
		t.Errorf("expected status %q, got %q", model.MediaStatusPending, status)
	}
}
