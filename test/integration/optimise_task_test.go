package integration

import (
	"bytes"
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/fhuszti/medias-ms-go/internal/db"
	"github.com/fhuszti/medias-ms-go/internal/migration"
	"github.com/fhuszti/medias-ms-go/internal/model"
	"github.com/fhuszti/medias-ms-go/internal/repository/mariadb"
	"github.com/fhuszti/medias-ms-go/internal/task"
	"github.com/fhuszti/medias-ms-go/test/testutil"
	"github.com/google/uuid"
)

func waitOptimised(t *testing.T, repo *mariadb.MediaRepository, id db.UUID, wantVariants bool) *model.Media {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for {
		out, err := repo.GetByID(context.Background(), id)
		if err != nil {
			t.Fatalf("GetByID: %v", err)
		}
		if out.Optimised && (!wantVariants || len(out.Variants) > 0) {
			return out
		}
		if time.Now().After(deadline) {
			t.Fatalf("timeout waiting for optimisation of %s", id)
		}
		time.Sleep(500 * time.Millisecond)
	}
}

func TestOptimiseTaskIntegration_SuccessPNG(t *testing.T) {
	ctx := context.Background()

	testDB, err := testutil.SetupTestDB()
	if err != nil {
		t.Fatalf("setup DB: %v", err)
	}
	defer testDB.Cleanup()
	dbConn := testDB.DB
	if err := migration.MigrateUp(dbConn); err != nil {
		t.Fatalf("could not run migrations: %v", err)
	}

	bCleanup, err := testutil.SetupTestBuckets(GlobalStrg)
	if err != nil {
		t.Fatalf("setup buckets: %v", err)
	}
	defer bCleanup()

	repo := mariadb.NewMediaRepository(dbConn)
	workerStop := testutil.StartWorker(&db.Database{dbConn}, GlobalStrg, RedisAddr)
	defer workerStop()

	id := db.UUID(uuid.MustParse("11111111-1111-1111-1111-111111111111"))
	objectKey := id.String() + ".png"
	width, height := 16, 32
	content := testutil.GeneratePNG(t, width, height)
	size := int64(len(content))
	mime := "image/png"
	m := &model.Media{
		ID:        id,
		ObjectKey: objectKey,
		Bucket:    "images",
		Status:    model.MediaStatusCompleted,
		MimeType:  &mime,
		SizeBytes: &size,
		Metadata:  model.Metadata{Width: width, Height: height},
	}
	if err := repo.Create(ctx, m); err != nil {
		t.Fatalf("insert media: %v", err)
	}
	if err := GlobalStrg.SaveFile(ctx, "images", objectKey, bytes.NewReader(content), size, map[string]string{"Content-Type": mime}); err != nil {
		t.Fatalf("upload file: %v", err)
	}

	dispatcher := task.NewDispatcher(RedisAddr, "")
	if err := dispatcher.EnqueueOptimiseMedia(ctx, id); err != nil {
		t.Fatalf("enqueue: %v", err)
	}

	out := waitOptimised(t, repo, id, true)

	if out.ObjectKey != id.String()+".webp" {
		t.Errorf("ObjectKey = %q; want %q", out.ObjectKey, id.String()+".webp")
	}
	if out.MimeType == nil || *out.MimeType != "image/webp" {
		t.Errorf("MimeType = %v; want image/webp", out.MimeType)
	}
	if len(out.Variants) != 1 {
		t.Fatalf("len(Variants) = %d; want 1", len(out.Variants))
	}
	v := out.Variants[0]
	if v.Width != 100 || v.Height != 200 {
		t.Errorf("variant dimensions = %dx%d; want 100x200", v.Width, v.Height)
	}
	exists, err := GlobalStrg.FileExists(ctx, "images", out.ObjectKey)
	if err != nil || !exists {
		t.Fatalf("optimised file missing: %v", err)
	}
	variantKey := fmt.Sprintf("variants/%s/%s_100.webp", id, id)
	ex, err := GlobalStrg.FileExists(ctx, "images", variantKey)
	if err != nil || !ex {
		t.Fatalf("variant file missing: %v", err)
	}
	oldExists, err := GlobalStrg.FileExists(ctx, "images", objectKey)
	if err != nil {
		t.Fatalf("check old file: %v", err)
	}
	if oldExists {
		t.Error("old file still exists after optimisation")
	}
}

func TestOptimiseTaskIntegration_SuccessWEBP(t *testing.T) {
	ctx := context.Background()
	testDB, err := testutil.SetupTestDB()
	if err != nil {
		t.Fatalf("setup DB: %v", err)
	}
	defer testDB.Cleanup()
	dbConn := testDB.DB
	if err := migration.MigrateUp(dbConn); err != nil {
		t.Fatalf("could not run migrations: %v", err)
	}
	bCleanup, err := testutil.SetupTestBuckets(GlobalStrg)
	if err != nil {
		t.Fatalf("setup buckets: %v", err)
	}
	defer bCleanup()
	repo := mariadb.NewMediaRepository(dbConn)
	workerStop := testutil.StartWorker(&db.Database{dbConn}, GlobalStrg, RedisAddr)
	defer workerStop()

	id := db.UUID(uuid.MustParse("22222222-2222-2222-2222-222222222222"))
	objectKey := id.String() + ".webp"
	width, height := 30, 60
	content := testutil.GenerateWebP(t, width, height)
	size := int64(len(content))
	mime := "image/webp"
	m := &model.Media{ID: id, ObjectKey: objectKey, Bucket: "images", Status: model.MediaStatusCompleted, MimeType: &mime, SizeBytes: &size, Metadata: model.Metadata{Width: width, Height: height}}
	if err := repo.Create(ctx, m); err != nil {
		t.Fatalf("insert media: %v", err)
	}
	if err := GlobalStrg.SaveFile(ctx, "images", objectKey, bytes.NewReader(content), size, map[string]string{"Content-Type": mime}); err != nil {
		t.Fatalf("upload file: %v", err)
	}

	dispatcher := task.NewDispatcher(RedisAddr, "")
	if err := dispatcher.EnqueueOptimiseMedia(ctx, id); err != nil {
		t.Fatalf("enqueue: %v", err)
	}

	out := waitOptimised(t, repo, id, true)
	if out.ObjectKey != objectKey { // should stay same
		t.Errorf("ObjectKey changed to %q", out.ObjectKey)
	}
	if out.MimeType == nil || *out.MimeType != "image/webp" {
		t.Errorf("MimeType = %v; want image/webp", out.MimeType)
	}
	if len(out.Variants) != 1 {
		t.Fatalf("len(Variants) = %d; want 1", len(out.Variants))
	}
	v := out.Variants[0]
	if v.Width != 100 || v.Height != 200 {
		t.Errorf("variant dimensions = %dx%d; want 100x200", v.Width, v.Height)
	}
	vKey := fmt.Sprintf("variants/%s/%s_100.webp", id, id)
	exists, err := GlobalStrg.FileExists(ctx, "images", vKey)
	if err != nil || !exists {
		t.Fatalf("variant file missing: %v", err)
	}
}

func TestOptimiseTaskIntegration_SuccessPDF(t *testing.T) {
	ctx := context.Background()
	testDB, err := testutil.SetupTestDB()
	if err != nil {
		t.Fatalf("setup DB: %v", err)
	}
	defer testDB.Cleanup()
	dbConn := testDB.DB
	if err := migration.MigrateUp(dbConn); err != nil {
		t.Fatalf("could not run migrations: %v", err)
	}
	bCleanup, err := testutil.SetupTestBuckets(GlobalStrg)
	if err != nil {
		t.Fatalf("setup buckets: %v", err)
	}
	defer bCleanup()
	repo := mariadb.NewMediaRepository(dbConn)
	workerStop := testutil.StartWorker(&db.Database{dbConn}, GlobalStrg, RedisAddr)
	defer workerStop()

	id := db.UUID(uuid.MustParse("33333333-3333-3333-3333-333333333333"))
	objectKey := id.String() + ".pdf"
	content := testutil.LoadPDF(t)
	size := int64(len(content))
	mime := "application/pdf"
	m := &model.Media{ID: id, ObjectKey: objectKey, Bucket: "docs", Status: model.MediaStatusCompleted, MimeType: &mime, SizeBytes: &size, Metadata: model.Metadata{PageCount: 4}}
	if err := repo.Create(ctx, m); err != nil {
		t.Fatalf("insert media: %v", err)
	}
	if err := GlobalStrg.SaveFile(ctx, "docs", objectKey, bytes.NewReader(content), size, map[string]string{"Content-Type": mime}); err != nil {
		t.Fatalf("upload file: %v", err)
	}

	dispatcher := task.NewDispatcher(RedisAddr, "")
	if err := dispatcher.EnqueueOptimiseMedia(ctx, id); err != nil {
		t.Fatalf("enqueue: %v", err)
	}

	out := waitOptimised(t, repo, id, false)
	if out.ObjectKey != objectKey {
		t.Errorf("ObjectKey changed to %q", out.ObjectKey)
	}
	if out.MimeType == nil || *out.MimeType != mime {
		t.Errorf("MimeType = %v; want %s", out.MimeType, mime)
	}
	if len(out.Variants) != 0 {
		t.Fatalf("variants count = %d; want 0", len(out.Variants))
	}
}

func TestOptimiseTaskIntegration_SuccessMarkdown(t *testing.T) {
	ctx := context.Background()
	testDB, err := testutil.SetupTestDB()
	if err != nil {
		t.Fatalf("setup DB: %v", err)
	}
	defer testDB.Cleanup()
	dbConn := testDB.DB
	if err := migration.MigrateUp(dbConn); err != nil {
		t.Fatalf("could not run migrations: %v", err)
	}
	bCleanup, err := testutil.SetupTestBuckets(GlobalStrg)
	if err != nil {
		t.Fatalf("setup buckets: %v", err)
	}
	defer bCleanup()
	repo := mariadb.NewMediaRepository(dbConn)
	workerStop := testutil.StartWorker(&db.Database{dbConn}, GlobalStrg, RedisAddr)
	defer workerStop()

	id := db.UUID(uuid.MustParse("44444444-4444-4444-4444-444444444444"))
	objectKey := id.String() + ".md"
	content := testutil.GenerateMarkdown()
	size := int64(len(content))
	mime := "text/markdown"
	m := &model.Media{ID: id, ObjectKey: objectKey, Bucket: "docs", Status: model.MediaStatusCompleted, MimeType: &mime, SizeBytes: &size}
	if err := repo.Create(ctx, m); err != nil {
		t.Fatalf("insert media: %v", err)
	}
	if err := GlobalStrg.SaveFile(ctx, "docs", objectKey, bytes.NewReader(content), size, map[string]string{"Content-Type": mime}); err != nil {
		t.Fatalf("upload file: %v", err)
	}

	dispatcher := task.NewDispatcher(RedisAddr, "")
	if err := dispatcher.EnqueueOptimiseMedia(ctx, id); err != nil {
		t.Fatalf("enqueue: %v", err)
	}

	out := waitOptimised(t, repo, id, false)
	if out.ObjectKey != objectKey {
		t.Errorf("ObjectKey changed to %q", out.ObjectKey)
	}
	if out.MimeType == nil || *out.MimeType != mime {
		t.Errorf("MimeType = %v; want %s", out.MimeType, mime)
	}
	if len(out.Variants) != 0 {
		t.Fatalf("variants count = %d; want 0", len(out.Variants))
	}
}

func TestOptimiseTaskIntegration_ErrorWrongStatus(t *testing.T) {
	ctx := context.Background()
	testDB, err := testutil.SetupTestDB()
	if err != nil {
		t.Fatalf("setup DB: %v", err)
	}
	defer testDB.Cleanup()
	dbConn := testDB.DB
	if err := migration.MigrateUp(dbConn); err != nil {
		t.Fatalf("could not run migrations: %v", err)
	}
	bCleanup, err := testutil.SetupTestBuckets(GlobalStrg)
	if err != nil {
		t.Fatalf("setup buckets: %v", err)
	}
	defer bCleanup()
	repo := mariadb.NewMediaRepository(dbConn)
	workerStop := testutil.StartWorker(&db.Database{dbConn}, GlobalStrg, RedisAddr)
	defer workerStop()

	id := db.UUID(uuid.MustParse("55555555-5555-5555-5555-555555555555"))
	objectKey := id.String() + ".png"
	content := testutil.GeneratePNG(t, 10, 10)
	size := int64(len(content))
	mime := "image/png"
	m := &model.Media{ID: id, ObjectKey: objectKey, Bucket: "images", Status: model.MediaStatusPending, MimeType: &mime, SizeBytes: &size}
	if err := repo.Create(ctx, m); err != nil {
		t.Fatalf("insert media: %v", err)
	}
	if err := GlobalStrg.SaveFile(ctx, "images", objectKey, bytes.NewReader(content), size, map[string]string{"Content-Type": mime}); err != nil {
		t.Fatalf("upload file: %v", err)
	}

	dispatcher := task.NewDispatcher(RedisAddr, "")
	if err := dispatcher.EnqueueOptimiseMedia(ctx, id); err != nil {
		t.Fatalf("enqueue: %v", err)
	}

	// wait a short period
	time.Sleep(3 * time.Second)
	out, err := repo.GetByID(ctx, id)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if out.Optimised {
		t.Error("unexpected optimisation")
	}
}

func TestOptimiseTaskIntegration_ErrorMissingFile(t *testing.T) {
	ctx := context.Background()
	testDB, err := testutil.SetupTestDB()
	if err != nil {
		t.Fatalf("setup DB: %v", err)
	}
	defer testDB.Cleanup()
	dbConn := testDB.DB
	if err := migration.MigrateUp(dbConn); err != nil {
		t.Fatalf("could not run migrations: %v", err)
	}
	bCleanup, err := testutil.SetupTestBuckets(GlobalStrg)
	if err != nil {
		t.Fatalf("setup buckets: %v", err)
	}
	defer bCleanup()
	repo := mariadb.NewMediaRepository(dbConn)
	workerStop := testutil.StartWorker(&db.Database{dbConn}, GlobalStrg, RedisAddr)
	defer workerStop()

	id := db.UUID(uuid.MustParse("66666666-6666-6666-6666-666666666666"))
	objectKey := id.String() + ".png"
	size := int64(100)
	mime := "image/png"
	m := &model.Media{ID: id, ObjectKey: objectKey, Bucket: "images", Status: model.MediaStatusCompleted, MimeType: &mime, SizeBytes: &size}
	if err := repo.Create(ctx, m); err != nil {
		t.Fatalf("insert media: %v", err)
	}
	// Note: file is not uploaded to storage

	dispatcher := task.NewDispatcher(RedisAddr, "")
	if err := dispatcher.EnqueueOptimiseMedia(ctx, id); err != nil {
		t.Fatalf("enqueue: %v", err)
	}

	time.Sleep(3 * time.Second)
	out, err := repo.GetByID(ctx, id)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if out.Optimised {
		t.Error("unexpected optimisation when file missing")
	}
}
