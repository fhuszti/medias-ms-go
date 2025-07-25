package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/fhuszti/medias-ms-go/internal/handler/api"
	"github.com/fhuszti/medias-ms-go/internal/migration"
	"github.com/fhuszti/medias-ms-go/internal/model"
	"github.com/fhuszti/medias-ms-go/internal/port"
	"github.com/fhuszti/medias-ms-go/internal/repository/mariadb"
	"github.com/fhuszti/medias-ms-go/internal/task"
	mediaSvc "github.com/fhuszti/medias-ms-go/internal/usecase/media"
	msuuid "github.com/fhuszti/medias-ms-go/internal/uuid"
	"github.com/fhuszti/medias-ms-go/test/testutil"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func setupUploadFinaliser(t *testing.T) (*mariadb.MediaRepository, port.UploadFinaliser, func()) {
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
	svc := mediaSvc.NewUploadFinaliser(repo, GlobalStrg, task.NewNoopDispatcher())

	cleanup := func() {
		_ = bCleanup()
		_ = testDB.Cleanup()
	}

	return repo, svc, cleanup
}

func TestFinaliseUploadIntegration_SuccessMarkdown(t *testing.T) {
	ctx := context.Background()

	mediaRepo, svc, cleanup := setupUploadFinaliser(t)
	defer cleanup()

	// Prepare media record and staging file
	id := msuuid.UUID(uuid.MustParse("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"))
	objectKey := id.String()
	destObjectKey := objectKey + ".md"
	content := testutil.GenerateMarkdown()

	m := &model.Media{
		ID:        id,
		ObjectKey: objectKey,
		Status:    model.MediaStatusPending,
	}
	if err := mediaRepo.Create(ctx, m); err != nil {
		t.Fatalf("insert media: %v", err)
	}

	// upload into "staging"
	if err := GlobalStrg.SaveFile(
		ctx,
		"staging",
		objectKey,
		bytes.NewReader(content),
		int64(len(content)),
		map[string]string{
			"Content-Type": "text/markdown",
		},
	); err != nil {
		t.Fatalf("upload to staging: %v", err)
	}

	if err := svc.FinaliseUpload(ctx, port.FinaliseUploadInput{
		ID:         id,
		DestBucket: "docs",
	}); err != nil {
		t.Fatalf("FinaliseUpload returned error: %v", err)
	}

	// Assert DB was updated
	fromDB, err := mediaRepo.GetByID(ctx, id)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if fromDB.Bucket != "docs" {
		t.Errorf("bucket should be 'docs', got %q", fromDB.Bucket)
	}
	if fromDB.Status != model.MediaStatusCompleted {
		t.Errorf("DB Status = %q; want %q", fromDB.Status, model.MediaStatusCompleted)
	}

	// Assert file moved to "docs" and absent from "staging"
	exists, err := GlobalStrg.FileExists(ctx, "docs", destObjectKey)
	if err != nil {
		t.Fatalf("checking dest FileExists: %v", err)
	}
	if !exists {
		t.Error("expected file in dest bucket, but it does not exist")
	}

	stillThere, err := GlobalStrg.FileExists(ctx, "staging", objectKey)
	if err != nil {
		t.Fatalf("checking staging FileExists: %v", err)
	}
	if stillThere {
		t.Error("expected staging file to be removed, but it still exists")
	}

	// Assert content round-trips
	rsc, err := GlobalStrg.GetFile(ctx, "docs", destObjectKey)
	if err != nil {
		t.Fatalf("GetFile on dest: %v", err)
	}
	defer rsc.Close()
	dataOut, err := io.ReadAll(rsc)
	if err != nil {
		t.Fatalf("reading dest file: %v", err)
	}
	if !bytes.Equal(dataOut, content) {
		t.Errorf("dest file content = %q; want %q", dataOut, content)
	}
}

func TestFinaliseUploadIntegration_SuccessImage(t *testing.T) {
	ctx := context.Background()

	mediaRepo, svc, cleanup := setupUploadFinaliser(t)
	defer cleanup()

	// Prepare a media record and staging file (PNG)
	id := msuuid.UUID(uuid.MustParse("bbbbbbbb-cccc-dddd-eeee-ffffffffffff"))
	objectKey := id.String()
	destObjectKey := objectKey + ".png"

	width, height := 16, 32
	content := testutil.GeneratePNG(t, width, height)

	m := &model.Media{
		ID:        id,
		ObjectKey: objectKey,
		Status:    model.MediaStatusPending,
	}
	if err := mediaRepo.Create(ctx, m); err != nil {
		t.Fatalf("insert media: %v", err)
	}

	// Upload into staging
	if err := GlobalStrg.SaveFile(
		ctx,
		"staging",
		objectKey,
		bytes.NewReader(content),
		int64(len(content)),
		map[string]string{"Content-Type": "image/png"},
	); err != nil {
		t.Fatalf("upload to staging: %v", err)
	}

	// Execute finalisation
	if err := svc.FinaliseUpload(ctx, port.FinaliseUploadInput{
		ID:         id,
		DestBucket: "images",
	}); err != nil {
		t.Fatalf("FinaliseUpload returned error: %v", err)
	}

	// Assert DB updated
	fromDB, err := mediaRepo.GetByID(ctx, id)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if fromDB.Bucket != "images" {
		t.Errorf("bucket should be 'images', got %q", fromDB.Bucket)
	}
	if fromDB.Status != model.MediaStatusCompleted {
		t.Errorf("DB Status = %q; want %q", fromDB.Status, model.MediaStatusCompleted)
	}
	if fromDB.SizeBytes == nil || *fromDB.SizeBytes != int64(len(content)) {
		t.Errorf("DB SizeBytes = %v; want %v", fromDB.SizeBytes, len(content))
	}
	if fromDB.MimeType == nil || *fromDB.MimeType != "image/png" {
		t.Errorf("DB MimeType = %q; want %q", *fromDB.MimeType, "image/png")
	}
	if fromDB.Metadata.Width != width {
		t.Errorf("Metadata.Width = %d; want %d", fromDB.Metadata.Width, width)
	}
	if fromDB.Metadata.Height != height {
		t.Errorf("Metadata.Height = %d; want %d", fromDB.Metadata.Height, height)
	}

	// Assert file moved to destination
	exists, err := GlobalStrg.FileExists(ctx, "images", destObjectKey)
	if err != nil {
		t.Fatalf("checking dest FileExists: %v", err)
	}
	if !exists {
		t.Error("expected file in dest bucket, but it does not exist")
	}

	// Staging should be cleaned up
	stillThere, err := GlobalStrg.FileExists(ctx, "staging", objectKey)
	if err != nil {
		t.Fatalf("checking staging FileExists: %v", err)
	}
	if stillThere {
		t.Error("expected staging file to be removed, but it still exists")
	}

	// Assert content round-trips
	rsc, err := GlobalStrg.GetFile(ctx, "images", destObjectKey)
	if err != nil {
		t.Fatalf("GetFile on dest: %v", err)
	}
	defer rsc.Close()
	dataOut, err := io.ReadAll(rsc)
	if err != nil {
		t.Fatalf("reading dest file: %v", err)
	}
	if !bytes.Equal(dataOut, content) {
		t.Errorf("dest file content = %d bytes; want %d bytes", len(dataOut), len(content))
	}
}

func TestFinaliseUploadIntegration_SuccessPDF(t *testing.T) {
	ctx := context.Background()

	mediaRepo, svc, cleanup := setupUploadFinaliser(t)
	defer cleanup()

	// Prepare media record and a staging file (PDF)
	id := msuuid.UUID(uuid.MustParse("cccccccc-dddd-eeee-ffff-000000000000"))
	objectKey := id.String()
	destObjectKey := objectKey + ".pdf"

	content := testutil.LoadPDF(t)

	m := &model.Media{
		ID:        id,
		ObjectKey: objectKey,
		Status:    model.MediaStatusPending,
	}
	if err := mediaRepo.Create(ctx, m); err != nil {
		t.Fatalf("insert media: %v", err)
	}

	// Upload into staging
	if err := GlobalStrg.SaveFile(
		ctx,
		"staging",
		objectKey,
		bytes.NewReader(content),
		int64(len(content)),
		map[string]string{"Content-Type": "application/pdf"},
	); err != nil {
		t.Fatalf("upload to staging: %v", err)
	}

	// Execute finalisation
	if err := svc.FinaliseUpload(ctx, port.FinaliseUploadInput{
		ID:         id,
		DestBucket: "docs",
	}); err != nil {
		t.Fatalf("FinaliseUpload returned error: %v", err)
	}

	// Assert DB updated
	fromDB, err := mediaRepo.GetByID(ctx, id)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if fromDB.Bucket != "docs" {
		t.Errorf("bucket should be 'docs', got %q", fromDB.Bucket)
	}
	if fromDB.Status != model.MediaStatusCompleted {
		t.Errorf("DB Status = %q; want %q", fromDB.Status, model.MediaStatusCompleted)
	}
	if fromDB.SizeBytes == nil || *fromDB.SizeBytes != int64(len(content)) {
		t.Errorf("DB SizeBytes = %v; want %v", fromDB.SizeBytes, len(content))
	}
	if fromDB.MimeType == nil || *fromDB.MimeType != "application/pdf" {
		t.Errorf("DB MimeType = %q; want %q", *fromDB.MimeType, "application/pdf")
	}
	if fromDB.Metadata.PageCount != 4 {
		t.Errorf("PageCount = %d; want %d", fromDB.Metadata.PageCount, 4)
	}

	// Assert file moved to destination
	exists, err := GlobalStrg.FileExists(ctx, "docs", destObjectKey)
	if err != nil {
		t.Fatalf("checking dest FileExists: %v", err)
	}
	if !exists {
		t.Error("expected file in dest bucket, but it does not exist")
	}

	// Staging should be cleaned up
	stillThere, err := GlobalStrg.FileExists(ctx, "staging", objectKey)
	if err != nil {
		t.Fatalf("checking staging FileExists: %v", err)
	}
	if stillThere {
		t.Error("expected staging file to be removed, but it still exists")
	}

	// Assert content round-trips
	rsc, err := GlobalStrg.GetFile(ctx, "docs", destObjectKey)
	if err != nil {
		t.Fatalf("GetFile on dest: %v", err)
	}
	defer rsc.Close()
	dataOut, err := io.ReadAll(rsc)
	if err != nil {
		t.Fatalf("reading dest file: %v", err)
	}
	if !bytes.Equal(dataOut, content) {
		t.Errorf("dest file content = %d bytes; want %d bytes", len(dataOut), len(content))
	}
}

func TestFinaliseUploadIntegration_Idempotency(t *testing.T) {
	ctx := context.Background()

	mediaRepo, svc, cleanup := setupUploadFinaliser(t)
	defer cleanup()

	// Prepare a Markdown payload in staging
	id := msuuid.UUID(uuid.MustParse("dddddddd-eeee-ffff-0000-111111111111"))
	objectKey := id.String()
	destObjectKey := objectKey + ".md"
	content := testutil.GenerateMarkdown()

	m := &model.Media{
		ID:        id,
		ObjectKey: objectKey,
		Status:    model.MediaStatusPending,
	}
	if err := mediaRepo.Create(ctx, m); err != nil {
		t.Fatalf("insert media: %v", err)
	}
	// Upload to staging
	if err := GlobalStrg.SaveFile(ctx, "staging", objectKey, bytes.NewReader(content), int64(len(content)), map[string]string{"Content-Type": "text/markdown"}); err != nil {
		t.Fatalf("upload to staging: %v", err)
	}

	// First call: expect success
	if err := svc.FinaliseUpload(ctx, port.FinaliseUploadInput{ID: id, DestBucket: "docs"}); err != nil {
		t.Fatalf("first FinaliseUpload error: %v", err)
	}

	out1, err := mediaRepo.GetByID(ctx, id)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if out1.Status != model.MediaStatusCompleted {
		t.Errorf("first call Status = %q; want %q", out1.Status, model.MediaStatusCompleted)
	}

	// Second call: should be no-op, return existing
	if err := svc.FinaliseUpload(ctx, port.FinaliseUploadInput{ID: id, DestBucket: "docs"}); err != nil {
		t.Fatalf("second FinaliseUpload error: %v", err)
	}

	out2, err := mediaRepo.GetByID(ctx, id)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if out2.Status != model.MediaStatusCompleted {
		t.Errorf("second call Status = %q; want %q", out2.Status, model.MediaStatusCompleted)
	}
	if out2.Bucket != out1.Bucket {
		t.Errorf("second call Bucket = %q; want %q", out2.Bucket, out1.Bucket)
	}
	if out2.ObjectKey != out1.ObjectKey {
		t.Errorf("second call ObjectKey = %q; want %q", out2.ObjectKey, out1.ObjectKey)
	}

	// Destination file exists
	exists, err := GlobalStrg.FileExists(ctx, "docs", destObjectKey)
	if err != nil {
		t.Fatalf("checking dest FileExists: %v", err)
	}
	if !exists {
		t.Error("expected file in dest bucket after idempotent calls, but it does not exist")
	}

	// Staging remains empty
	stillThere, err := GlobalStrg.FileExists(ctx, "staging", objectKey)
	if err != nil {
		t.Fatalf("checking staging FileExists: %v", err)
	}
	if stillThere {
		t.Error("expected staging file to be removed after idempotency, but it still exists")
	}

	// Round-trip content still same
	rsc, err := GlobalStrg.GetFile(ctx, "docs", destObjectKey)
	if err != nil {
		t.Fatalf("GetFile on dest: %v", err)
	}
	defer rsc.Close()
	dataOut, err := io.ReadAll(rsc)
	if err != nil {
		t.Fatalf("reading dest file: %v", err)
	}
	if !bytes.Equal(dataOut, content) {
		t.Errorf("dest content changed after idempotent calls: got %d bytes; want %d bytes", len(dataOut), len(content))
	}
}

func TestFinaliseUploadIntegration_ErrorFileSize(t *testing.T) {
	ctx := context.Background()

	mediaRepo, svc, cleanup := setupUploadFinaliser(t)
	defer cleanup()

	// Prepare an undersized Markdown file
	id := msuuid.UUID(uuid.MustParse("eeeeeeee-ffff-0000-1111-222222222222"))
	objectKey := id.String()
	destObjectKey := objectKey + ".md"
	// content length = MinFileSize - 1
	content := bytes.Repeat([]byte("x"), mediaSvc.MinFileSize-1)

	m := &model.Media{
		ID:        id,
		ObjectKey: objectKey,
		Status:    model.MediaStatusPending,
	}
	if err := mediaRepo.Create(ctx, m); err != nil {
		t.Fatalf("insert media: %v", err)
	}

	// Upload to staging
	if err := GlobalStrg.SaveFile(
		ctx,
		"staging",
		objectKey,
		bytes.NewReader(content),
		int64(len(content)),
		map[string]string{"Content-Type": "text/markdown"},
	); err != nil {
		t.Fatalf("upload to staging: %v", err)
	}

	// Attempt finalisation: expect "too small" error
	err := svc.FinaliseUpload(ctx, port.FinaliseUploadInput{ID: id, DestBucket: "docs"})
	if err == nil {
		t.Fatalf("expected error for too small file, got nil")
	}
	if !strings.Contains(err.Error(), "too small") {
		t.Errorf("error = %q; want substring 'too small'", err.Error())
	}

	// Staging file should be cleaned up
	stillStaged, err := GlobalStrg.FileExists(ctx, "staging", objectKey)
	if err != nil {
		t.Fatalf("checking staging FileExists: %v", err)
	}
	if stillStaged {
		t.Error("expected staging file to be removed after failure, but it still exists")
	}

	// DB record should be marked Failed with the appropriate message
	fromDB, err := mediaRepo.GetByID(ctx, id)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if fromDB.Status != model.MediaStatusFailed {
		t.Errorf("DB Status = %q; want %q", fromDB.Status, model.MediaStatusFailed)
	}
	if fromDB.FailureMessage == nil || !strings.Contains(*fromDB.FailureMessage, "too small") {
		t.Errorf("FailureMessage = %v; want to contain 'too small'", fromDB.FailureMessage)
	}

	// No file should appear in the destination bucket
	exists, err := GlobalStrg.FileExists(ctx, "docs", destObjectKey)
	if err != nil {
		t.Fatalf("checking dest FileExists: %v", err)
	}
	if exists {
		t.Error("expected no file in dest bucket after failure, but found one")
	}
}

func TestFinaliseUploadIntegration_ErrorInvalidBucket(t *testing.T) {
	r := chi.NewRouter()
	allowed := []string{"images", "docs"}
	r.With(api.WithID()).
		Post("/medias/finalise_upload/{id}", api.FinaliseUploadHandler(nil, allowed))

	body := strings.NewReader(`{"dest_bucket":"not-a-bucket"}`)
	req := httptest.NewRequest("POST", "/medias/finalise_upload/aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee", body)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d; want %d", rec.Code, http.StatusBadRequest)
	}

	var resp errorResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("could not decode JSON: %v", err)
	}
	wantMsg := `destination bucket "not-a-bucket" does not exist`
	if resp.Error != wantMsg {
		t.Errorf("error = %q; want %q", resp.Error, wantMsg)
	}
}
