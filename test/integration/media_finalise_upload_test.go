package integration

import (
	"bytes"
	"context"
	"github.com/fhuszti/medias-ms-go/internal/db"
	"github.com/fhuszti/medias-ms-go/internal/migration"
	"github.com/fhuszti/medias-ms-go/internal/model"
	"github.com/fhuszti/medias-ms-go/internal/repository/mariadb"
	mediaService "github.com/fhuszti/medias-ms-go/internal/usecase/media"
	"github.com/fhuszti/medias-ms-go/test/testutil"
	"github.com/google/uuid"
	"io"
	"reflect"
	"strings"
	"testing"
)

func TestFinaliseUploadIntegration(t *testing.T) {
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
	defer func() {
		if err := tb.Cleanup(); err != nil {
			t.Fatalf("cleanup buckets: %v", err)
		}
	}()

	mediaRepo := mariadb.NewMediaRepository(database)
	stgStrg, err := tb.StrgClient.WithBucket("staging")
	if err != nil {
		t.Fatalf("failed to initialise bucket 'staging': %v", err)
	}
	getDestStrg := func(bucket string) (mediaService.Storage, error) {
		return tb.StrgClient.WithBucket(bucket)
	}
	svc := mediaService.NewUploadFinaliser(mediaRepo, stgStrg, getDestStrg)

	id := db.UUID(uuid.MustParse("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"))
	objectKey := id.String()
	destObjectKey := objectKey + ".md"
	content := []byte("# Hello E2E Test" + strings.Repeat(".", 1024))
	prepareDataForTest(id, objectKey, content, ctx, t, mediaRepo, stgStrg)

	out, err := svc.FinaliseUpload(ctx, mediaService.FinaliseUploadInput{
		ID:         id,
		DestBucket: "images",
	})
	if err != nil {
		t.Fatalf("FinaliseUpload returned error: %v", err)
	}

	// Assert returned media
	if out.ID != id {
		t.Errorf("returned ID = %v; want %v", out.ID, id)
	}
	if out.Bucket != "images" {
		t.Errorf("bucket should be 'images', got %q", out.Bucket)
	}
	if out.Status != model.MediaStatusCompleted {
		t.Errorf("returned Status = %q; want %q", out.Status, model.MediaStatusCompleted)
	}
	if out.SizeBytes == nil || *out.SizeBytes != int64(len(content)) {
		t.Errorf("returned SizeBytes = %v; want %v", out.SizeBytes, len(content))
	}
	if out.MimeType == nil || *out.MimeType != "text/markdown" {
		t.Errorf("returned MimeType = %q; want %q", *out.MimeType, "text/markdown")
	}
	if reflect.DeepEqual(out.Metadata, model.Metadata{}) {
		t.Errorf("expected non-empty Metadata struct, got %+v", out.Metadata)
	}

	// Assert DB was updated
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

	// Assert file moved to "images" and absent from "staging"
	destStrg, err := getDestStrg("images")
	if err != nil {
		t.Fatalf("init dest bucket: %v", err)
	}
	exists, err := destStrg.FileExists(ctx, destObjectKey)
	if err != nil {
		t.Fatalf("checking dest FileExists: %v", err)
	}
	if !exists {
		t.Error("expected file in dest bucket, but it does not exist")
	}

	stillThere, err := stgStrg.FileExists(ctx, objectKey)
	if err != nil {
		t.Fatalf("checking staging FileExists: %v", err)
	}
	if stillThere {
		t.Error("expected staging file to be removed, but it still exists")
	}

	// Assert content round-trips
	rc, err := destStrg.GetFile(ctx, destObjectKey)
	if err != nil {
		t.Fatalf("GetFile on dest: %v", err)
	}
	defer rc.Close()
	dataOut, err := io.ReadAll(rc)
	if err != nil {
		t.Fatalf("reading dest file: %v", err)
	}
	if !bytes.Equal(dataOut, content) {
		t.Errorf("dest file content = %q; want %q", dataOut, content)
	}
}

func prepareDataForTest(id db.UUID, objectKey string, content []byte, ctx context.Context, t *testing.T, mediaRepo *mariadb.MediaRepository, stgStrg mediaService.Storage) {
	m := &model.Media{
		ID:        id,
		ObjectKey: objectKey,
		Status:    model.MediaStatusPending,
	}
	if err := mediaRepo.Create(ctx, m); err != nil {
		t.Fatalf("insert media: %v", err)
	}

	// upload into "staging"
	if err := stgStrg.SaveFile(
		ctx,
		objectKey,
		bytes.NewReader(content),
		int64(len(content)),
		map[string]string{
			"Content-Type": "text/markdown",
		},
	); err != nil {
		t.Fatalf("upload to staging: %v", err)
	}
}
