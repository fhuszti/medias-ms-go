package integration

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/fhuszti/medias-ms-go/internal/migration"
	"github.com/fhuszti/medias-ms-go/internal/model"
	"github.com/fhuszti/medias-ms-go/internal/port"
	"github.com/fhuszti/medias-ms-go/internal/repository/mariadb"
	"github.com/fhuszti/medias-ms-go/internal/task"
	mediaSvc "github.com/fhuszti/medias-ms-go/internal/usecase/media"
	msuuid "github.com/fhuszti/medias-ms-go/internal/uuid"
	"github.com/fhuszti/medias-ms-go/test/testutil"
	"github.com/google/uuid"
)

func setupUpdateFinaliser(t *testing.T) (*mariadb.MediaRepository, port.UpdateFinaliser, func()) {
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
	svc := mediaSvc.NewUpdateFinaliser(repo, GlobalStrg, task.NewNoopDispatcher())

	cleanup := func() {
		_ = bCleanup()
		_ = testDB.Cleanup()
	}

	return repo, svc, cleanup
}

func TestFinaliseUpdateIntegration_SuccessImage(t *testing.T) {
	ctx := context.Background()

	mediaRepo, svc, cleanup := setupUpdateFinaliser(t)
	defer cleanup()

	id := msuuid.UUID(uuid.MustParse("11111111-2222-3333-4444-555555555555"))
	version := uint16(1)
	oldObjectKey := "old-image.webp"
	oldBucket := "images"

	variants := model.Variants{
		{ObjectKey: fmt.Sprintf("variants/%s/old_50.webp", id), SizeBytes: 10, Width: 50, Height: 50},
		{ObjectKey: fmt.Sprintf("variants/%s/old_100.webp", id), SizeBytes: 20, Width: 100, Height: 100},
	}

	m := &model.Media{
		ID:               id,
		ObjectKey:        oldObjectKey,
		Bucket:           oldBucket,
		OriginalFilename: "old-image.webp",
		Version:          version,
		Status:           model.MediaStatusCompleted,
		Optimised:        true,
		Variants:         variants,
	}
	if err := mediaRepo.Create(ctx, m); err != nil {
		t.Fatalf("insert media: %v", err)
	}

	oldContent := testutil.GeneratePNG(t, 8, 8)
	if err := GlobalStrg.SaveFile(ctx, oldBucket, oldObjectKey, bytes.NewReader(oldContent), int64(len(oldContent)), map[string]string{"Content-Type": "image/webp"}); err != nil {
		t.Fatalf("upload old object: %v", err)
	}
	for _, v := range variants {
		if err := GlobalStrg.SaveFile(ctx, oldBucket, v.ObjectKey, bytes.NewReader(oldContent), int64(len(oldContent)), map[string]string{"Content-Type": "image/webp"}); err != nil {
			t.Fatalf("upload variant %q: %v", v.ObjectKey, err)
		}
	}

	nextVersion := version + 1
	stagingKey := fmt.Sprintf("%s-v%d", id, nextVersion)
	newContent := testutil.GeneratePNG(t, 16, 32)
	if err := GlobalStrg.SaveFile(ctx, "staging", stagingKey, bytes.NewReader(newContent), int64(len(newContent)), map[string]string{"Content-Type": "image/png"}); err != nil {
		t.Fatalf("upload to staging: %v", err)
	}

	if err := svc.FinaliseUpdate(ctx, port.FinaliseUpdateInput{ID: id, DestBucket: "images"}); err != nil {
		t.Fatalf("FinaliseUpdate returned error: %v", err)
	}

	fromDB, err := mediaRepo.GetByID(ctx, id)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if fromDB.Status != model.MediaStatusCompleted {
		t.Errorf("Status = %q; want %q", fromDB.Status, model.MediaStatusCompleted)
	}
	if fromDB.Version != nextVersion {
		t.Errorf("Version = %d; want %d", fromDB.Version, nextVersion)
	}
	if fromDB.Bucket != "images" {
		t.Errorf("Bucket = %q; want %q", fromDB.Bucket, "images")
	}
	newObjectKey := fmt.Sprintf("%s.png", stagingKey)
	if fromDB.ObjectKey != newObjectKey {
		t.Errorf("ObjectKey = %q; want %q", fromDB.ObjectKey, newObjectKey)
	}
	if fromDB.Optimised {
		t.Error("expected Optimised to be false after update")
	}
	if fromDB.FailureMessage != nil {
		t.Errorf("FailureMessage = %v; want nil", *fromDB.FailureMessage)
	}
	if len(fromDB.Variants) != 0 {
		t.Errorf("Variants length = %d; want 0", len(fromDB.Variants))
	}
	if fromDB.SizeBytes == nil || *fromDB.SizeBytes != int64(len(newContent)) {
		t.Errorf("SizeBytes = %v; want %d", fromDB.SizeBytes, len(newContent))
	}
	if fromDB.MimeType == nil || *fromDB.MimeType != "image/png" {
		t.Errorf("MimeType = %v; want %q", fromDB.MimeType, "image/png")
	}
	if fromDB.Metadata.Width != 16 {
		t.Errorf("Metadata.Width = %d; want %d", fromDB.Metadata.Width, 16)
	}
	if fromDB.Metadata.Height != 32 {
		t.Errorf("Metadata.Height = %d; want %d", fromDB.Metadata.Height, 32)
	}

	// New object is present
	exists, err := GlobalStrg.FileExists(ctx, "images", newObjectKey)
	if err != nil {
		t.Fatalf("checking dest FileExists: %v", err)
	}
	if !exists {
		t.Error("expected updated object in destination bucket, but it does not exist")
	}

	// Staging should be cleaned up
	stillThere, err := GlobalStrg.FileExists(ctx, "staging", stagingKey)
	if err != nil {
		t.Fatalf("checking staging FileExists: %v", err)
	}
	if stillThere {
		t.Error("expected staging file to be removed, but it still exists")
	}

	// Previous object and variants should be removed
	prevExists, err := GlobalStrg.FileExists(ctx, oldBucket, oldObjectKey)
	if err != nil {
		t.Fatalf("checking previous FileExists: %v", err)
	}
	if prevExists {
		t.Error("expected previous object to be removed, but it still exists")
	}
	for _, v := range variants {
		ex, err := GlobalStrg.FileExists(ctx, oldBucket, v.ObjectKey)
		if err != nil {
			t.Fatalf("checking variant FileExists for %q: %v", v.ObjectKey, err)
		}
		if ex {
			t.Errorf("expected variant %q to be removed, but it still exists", v.ObjectKey)
		}
	}

	// Content round-trips
	rsc, err := GlobalStrg.GetFile(ctx, "images", newObjectKey)
	if err != nil {
		t.Fatalf("GetFile on dest: %v", err)
	}
	defer rsc.Close()
	dataOut, err := io.ReadAll(rsc)
	if err != nil {
		t.Fatalf("reading dest file: %v", err)
	}
	if !bytes.Equal(dataOut, newContent) {
		t.Errorf("dest file content = %d bytes; want %d bytes", len(dataOut), len(newContent))
	}
}

func TestFinaliseUpdateIntegration_ErrorFileSize(t *testing.T) {
	ctx := context.Background()

	mediaRepo, svc, cleanup := setupUpdateFinaliser(t)
	defer cleanup()

	id := msuuid.UUID(uuid.MustParse("66666666-7777-8888-9999-000000000000"))
	version := uint16(1)
	oldObjectKey := "existing.png"
	bucket := "images"

	variants := model.Variants{
		{ObjectKey: fmt.Sprintf("variants/%s/existing_50.webp", id), SizeBytes: 10, Width: 50, Height: 50},
	}

	m := &model.Media{
		ID:               id,
		ObjectKey:        oldObjectKey,
		Bucket:           bucket,
		OriginalFilename: "existing.png",
		Version:          version,
		Status:           model.MediaStatusCompleted,
		Optimised:        true,
		Variants:         variants,
	}
	if err := mediaRepo.Create(ctx, m); err != nil {
		t.Fatalf("insert media: %v", err)
	}

	oldContent := testutil.GeneratePNG(t, 8, 8)
	if err := GlobalStrg.SaveFile(ctx, bucket, oldObjectKey, bytes.NewReader(oldContent), int64(len(oldContent)), map[string]string{"Content-Type": "image/png"}); err != nil {
		t.Fatalf("upload old object: %v", err)
	}
	for _, v := range variants {
		if err := GlobalStrg.SaveFile(ctx, bucket, v.ObjectKey, bytes.NewReader(oldContent), int64(len(oldContent)), map[string]string{"Content-Type": "image/png"}); err != nil {
			t.Fatalf("upload variant %q: %v", v.ObjectKey, err)
		}
	}

	stagingKey := fmt.Sprintf("%s-v%d", id, version+1)
	content := bytes.Repeat([]byte("x"), mediaSvc.MinFileSize-1)
	if err := GlobalStrg.SaveFile(ctx, "staging", stagingKey, bytes.NewReader(content), int64(len(content)), map[string]string{"Content-Type": "image/png"}); err != nil {
		t.Fatalf("upload to staging: %v", err)
	}

	err := svc.FinaliseUpdate(ctx, port.FinaliseUpdateInput{ID: id, DestBucket: bucket})
	if err == nil {
		t.Fatalf("expected error for too small file, got nil")
	}
	if !strings.Contains(err.Error(), "too small") {
		t.Errorf("error = %q; want substring 'too small'", err.Error())
	}

	// Staging should be cleaned up
	stillThere, err := GlobalStrg.FileExists(ctx, "staging", stagingKey)
	if err != nil {
		t.Fatalf("checking staging FileExists: %v", err)
	}
	if stillThere {
		t.Error("expected staging file to be removed after failure, but it still exists")
	}

	fromDB, err := mediaRepo.GetByID(ctx, id)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if fromDB.Status != model.MediaStatusFailed {
		t.Errorf("Status = %q; want %q", fromDB.Status, model.MediaStatusFailed)
	}
	if fromDB.FailureMessage == nil || !strings.Contains(*fromDB.FailureMessage, "too small") {
		t.Errorf("FailureMessage = %v; want to contain 'too small'", fromDB.FailureMessage)
	}

	// Previous object and variants should remain
	prevExists, err := GlobalStrg.FileExists(ctx, bucket, oldObjectKey)
	if err != nil {
		t.Fatalf("checking previous FileExists: %v", err)
	}
	if !prevExists {
		t.Error("expected previous object to remain after failure, but it was removed")
	}
	for _, v := range variants {
		ex, err := GlobalStrg.FileExists(ctx, bucket, v.ObjectKey)
		if err != nil {
			t.Fatalf("checking variant FileExists for %q: %v", v.ObjectKey, err)
		}
		if !ex {
			t.Errorf("expected variant %q to remain after failure, but it was removed", v.ObjectKey)
		}
	}

	// No new destination file should appear
	newObjectKey := fmt.Sprintf("%s.png", stagingKey)
	exists, err := GlobalStrg.FileExists(ctx, bucket, newObjectKey)
	if err != nil {
		t.Fatalf("checking dest FileExists: %v", err)
	}
	if exists {
		t.Error("expected no new file in destination bucket after failure, but found one")
	}
}

func TestFinaliseUpdateIntegration_ErrorWrongStatus(t *testing.T) {
	ctx := context.Background()

	mediaRepo, svc, cleanup := setupUpdateFinaliser(t)
	defer cleanup()

	id := msuuid.UUID(uuid.MustParse("99999999-aaaa-bbbb-cccc-dddddddddddd"))
	version := uint16(1)
	objectKey := "pending.png"

	m := &model.Media{
		ID:               id,
		ObjectKey:        objectKey,
		Bucket:           "images",
		OriginalFilename: "pending.png",
		Version:          version,
		Status:           model.MediaStatusPending,
	}
	if err := mediaRepo.Create(ctx, m); err != nil {
		t.Fatalf("insert media: %v", err)
	}

	stagingKey := fmt.Sprintf("%s-v%d", id, version+1)
	content := testutil.GeneratePNG(t, 12, 12)
	if err := GlobalStrg.SaveFile(ctx, "staging", stagingKey, bytes.NewReader(content), int64(len(content)), map[string]string{"Content-Type": "image/png"}); err != nil {
		t.Fatalf("upload to staging: %v", err)
	}

	err := svc.FinaliseUpdate(ctx, port.FinaliseUpdateInput{ID: id, DestBucket: "images"})
	if err == nil {
		t.Fatalf("expected error for wrong status, got nil")
	}
	if !strings.Contains(err.Error(), "media status should be 'completed'") {
		t.Errorf("error = %q; want substring %q", err.Error(), "media status should be 'completed'")
	}

	fromDB, err := mediaRepo.GetByID(ctx, id)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if fromDB.Status != model.MediaStatusPending {
		t.Errorf("Status = %q; want %q", fromDB.Status, model.MediaStatusPending)
	}
	if fromDB.FailureMessage != nil {
		t.Errorf("FailureMessage = %v; want nil", fromDB.FailureMessage)
	}

	stillThere, err := GlobalStrg.FileExists(ctx, "staging", stagingKey)
	if err != nil {
		t.Fatalf("checking staging FileExists: %v", err)
	}
	if !stillThere {
		t.Error("expected staging file to remain after wrong status, but it was removed")
	}
}
