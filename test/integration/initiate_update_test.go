package integration

import (
	"context"
	"errors"
	"net/url"
	"strings"
	"testing"

	"github.com/fhuszti/medias-ms-go/internal/migration"
	"github.com/fhuszti/medias-ms-go/internal/model"
	"github.com/fhuszti/medias-ms-go/internal/port"
	"github.com/fhuszti/medias-ms-go/internal/repository/mariadb"
	mediaSvc "github.com/fhuszti/medias-ms-go/internal/usecase/media"
	msuuid "github.com/fhuszti/medias-ms-go/internal/uuid"
	"github.com/fhuszti/medias-ms-go/test/testutil"
	"github.com/google/uuid"
)

func setupUpdateInitiator(t *testing.T) (*mariadb.MediaRepository, port.UpdateInitiator, func()) {
	t.Helper()

	testDB, err := testutil.SetupTestDB()
	if err != nil {
		t.Fatalf("setup DB: %v", err)
	}
	dbConn := testDB.DB
	if err := migration.MigrateUp(dbConn); err != nil {
		t.Fatalf("could not run migrations: %v", err)
	}

	bCleanup, err := testutil.SetupTestBuckets(GlobalStrg)
	if err != nil {
		t.Fatalf("setup buckets: %v", err)
	}

	repo := mariadb.NewMediaRepository(dbConn)
	svc := mediaSvc.NewUpdateInitiator(repo, GlobalStrg)

	cleanup := func() {
		_ = bCleanup()
		_ = testDB.Cleanup()
	}

	return repo, svc, cleanup
}

func TestInitiateUpdateIntegration_Success(t *testing.T) {
	ctx := context.Background()

	mediaRepo, svc, cleanup := setupUpdateInitiator(t)
	defer cleanup()

	id := msuuid.UUID(uuid.MustParse("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"))
	version := uint16(3)
	objectKey := id.String() + "-v3.png"

	m := &model.Media{
		ID:               id,
		ObjectKey:        objectKey,
		Bucket:           "images",
		OriginalFilename: "original.png",
		Version:          version,
		Status:           model.MediaStatusCompleted,
		MimeType:         ptrString("image/png"),
		SizeBytes:        ptrInt64(123),
		Metadata:         model.Metadata{Width: 12, Height: 34},
		Optimised:        true,
	}
	if err := mediaRepo.Create(ctx, m); err != nil {
		t.Fatalf("insert media: %v", err)
	}

	out, err := svc.InitiateUpdate(ctx, port.InitiateUpdateInput{ID: id, Name: "new.png"})
	if err != nil {
		t.Fatalf("InitiateUpdate returned error: %v", err)
	}
	if out.ID != id {
		t.Errorf("ID = %q; want %q", out.ID, id)
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
	bucketName, key := parts[0], parts[1]
	if bucketName != "staging" {
		t.Errorf("expected bucket 'staging', got %q", bucketName)
	}
	wantKey := id.String() + "-v4"
	if key != wantKey {
		t.Errorf("expected objectKey %q, got %q", wantKey, key)
	}

	// DB record should be unchanged
	fromDB, err := mediaRepo.GetByID(ctx, id)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if fromDB.ObjectKey != objectKey {
		t.Errorf("ObjectKey = %q; want %q", fromDB.ObjectKey, objectKey)
	}
	if fromDB.Version != version {
		t.Errorf("Version = %d; want %d", fromDB.Version, version)
	}
	if fromDB.Bucket != "images" {
		t.Errorf("Bucket = %q; want %q", fromDB.Bucket, "images")
	}
	if fromDB.Status != model.MediaStatusCompleted {
		t.Errorf("Status = %q; want %q", fromDB.Status, model.MediaStatusCompleted)
	}

	// Staging object should not exist yet
	exists, err := GlobalStrg.FileExists(ctx, "staging", wantKey)
	if err != nil {
		t.Fatalf("checking staging FileExists: %v", err)
	}
	if exists {
		t.Error("expected no staging file yet, but it exists")
	}
}

func TestInitiateUpdateIntegration_NotFound(t *testing.T) {
	ctx := context.Background()

	_, svc, cleanup := setupUpdateInitiator(t)
	defer cleanup()

	missingID := msuuid.UUID(uuid.MustParse("ffffffff-eeee-dddd-cccc-bbbbbbbbbbbb"))
	_, err := svc.InitiateUpdate(ctx, port.InitiateUpdateInput{ID: missingID, Name: "new.png"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, mediaSvc.ErrObjectNotFound) {
		t.Fatalf("expected ErrObjectNotFound, got %v", err)
	}
}
