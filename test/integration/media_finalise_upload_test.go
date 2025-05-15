package integration

import (
	"context"
	"github.com/fhuszti/medias-ms-go/internal/db"
	"github.com/fhuszti/medias-ms-go/internal/migration"
	"github.com/fhuszti/medias-ms-go/internal/repository/mariadb"
	mediaService "github.com/fhuszti/medias-ms-go/internal/usecase/media"
	"github.com/fhuszti/medias-ms-go/test/testutil"
	"github.com/google/uuid"
	"testing"
)

func TestFinaliseUploadIntegration(t *testing.T) {
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
	getDestBucket := func(bucket string) (mediaService.Storage, error) {
		return tb.StrgClient.WithBucket(bucket)
	}
	svc := mediaService.NewUploadFinaliser(mediaRepo, stgStrg, getDestBucket)

	in := mediaService.FinaliseUploadInput{
		ID:         db.UUID(uuid.MustParse("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee")),
		DestBucket: "images",
	}

	media, err := svc.FinaliseUpload(context.Background(), in)
	if err != nil {
		t.Fatalf("FinaliseUpload returned error: %v", err)
	}

	if media.ID == db.UUID(uuid.Nil) {
		t.Fatal("expected non-empty ID")
	}
}
