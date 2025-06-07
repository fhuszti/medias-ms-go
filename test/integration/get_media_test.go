package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/fhuszti/medias-ms-go/internal/cache"
	"github.com/fhuszti/medias-ms-go/internal/db"
	"github.com/fhuszti/medias-ms-go/internal/handler/api"
	"github.com/fhuszti/medias-ms-go/internal/migration"
	"github.com/fhuszti/medias-ms-go/internal/model"
	"github.com/fhuszti/medias-ms-go/internal/repository/mariadb"
	mediaSvc "github.com/fhuszti/medias-ms-go/internal/usecase/media"
	"github.com/fhuszti/medias-ms-go/test/testutil"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestGetMediaIntegration_SuccessMarkdown(t *testing.T) {
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

	bCleanup, err := testutil.SetupTestBuckets(GlobalStrg)
	if err != nil {
		t.Fatalf("setup buckets: %v", err)
	}
	defer bCleanup()

	mediaRepo := mariadb.NewMediaRepository(database)
	ca := cache.NewNoop()
	svc := mediaSvc.NewMediaGetter(mediaRepo, ca, GlobalStrg)

	id := db.UUID(uuid.MustParse("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"))
	objectKey := id.String() + ".md"
	bucket := "docs"
	content := testutil.GenerateMarkdown()
	meta := model.Metadata{WordCount: 23, HeadingCount: 3, LinkCount: 2}

	m := &model.Media{
		ID:        id,
		ObjectKey: objectKey,
		Bucket:    bucket,
		Status:    model.MediaStatusCompleted,
		Metadata:  meta,
		SizeBytes: ptrInt64(int64(len(content))),
		MimeType:  ptrString("text/markdown"),
	}
	if err := mediaRepo.Create(ctx, m); err != nil {
		t.Fatalf("insert media: %v", err)
	}

	if err := GlobalStrg.SaveFile(ctx, bucket, objectKey, bytes.NewReader(content), int64(len(content)), map[string]string{
		"Content-Type": "text/markdown",
	}); err != nil {
		t.Fatalf("upload to %q bucket: %v", bucket, err)
	}

	out, err := svc.GetMedia(ctx, mediaSvc.GetMediaInput{ID: id})
	if err != nil {
		t.Fatalf("GetMedia returned error: %v", err)
	}

	// Assert the output
	if out.URL == "" {
		t.Errorf("expected non-empty URL, got %q", out.URL)
	}
	if !strings.Contains(out.URL, objectKey) {
		t.Errorf("URL = %q; want to contain %q", out.URL, objectKey)
	}
	// ValidUntil should be in the future but within the next 3h
	if time.Until(out.ValidUntil) <= 0 || time.Until(out.ValidUntil) > 3*time.Hour {
		t.Errorf("ValidUntil = %v; want within next 3h", out.ValidUntil)
	}
	if out.Metadata.MimeType != "text/markdown" {
		t.Errorf("MimeType = %q; want %q", out.Metadata.MimeType, "text/markdown")
	}
	if out.Metadata.SizeBytes != int64(len(content)) {
		t.Errorf("SizeBytes = %d; want %d", out.Metadata.SizeBytes, len(content))
	}
	if out.Metadata.WordCount != 23 {
		t.Errorf("WordCount = %d; want %d", out.Metadata.WordCount, 23)
	}
	if out.Metadata.HeadingCount != 3 {
		t.Errorf("HeadingCount = %d; want %d", out.Metadata.HeadingCount, 3)
	}
	if out.Metadata.LinkCount != 2 {
		t.Errorf("LinkCount = %d; want %d", out.Metadata.LinkCount, 2)
	}
	// documents should have no variants
	if len(out.Variants) != 0 {
		t.Errorf("Variants length = %d; want 0 for documents", len(out.Variants))
	}
}

func TestGetMediaIntegration_SuccessPDF(t *testing.T) {
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

	bCleanup, err := testutil.SetupTestBuckets(GlobalStrg)
	if err != nil {
		t.Fatalf("setup buckets: %v", err)
	}
	defer bCleanup()

	mediaRepo := mariadb.NewMediaRepository(database)
	ca := cache.NewNoop()
	svc := mediaSvc.NewMediaGetter(mediaRepo, ca, GlobalStrg)

	id := db.UUID(uuid.MustParse("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"))
	objectKey := id.String() + ".md"
	bucket := "docs"
	content := testutil.LoadPDF(t)
	meta := model.Metadata{PageCount: 4}

	m := &model.Media{
		ID:        id,
		ObjectKey: objectKey,
		Bucket:    bucket,
		Status:    model.MediaStatusCompleted,
		Metadata:  meta,
		SizeBytes: ptrInt64(int64(len(content))),
		MimeType:  ptrString("application/pdf"),
	}
	if err := mediaRepo.Create(ctx, m); err != nil {
		t.Fatalf("insert media: %v", err)
	}

	if err := GlobalStrg.SaveFile(ctx, bucket, objectKey, bytes.NewReader(content), int64(len(content)), map[string]string{
		"Content-Type": "application/pdf",
	}); err != nil {
		t.Fatalf("upload to %q bucket: %v", bucket, err)
	}

	out, err := svc.GetMedia(ctx, mediaSvc.GetMediaInput{ID: id})
	if err != nil {
		t.Fatalf("GetMedia returned error: %v", err)
	}

	// Assert the output
	if out.URL == "" {
		t.Errorf("expected non-empty URL, got %q", out.URL)
	}
	if !strings.Contains(out.URL, objectKey) {
		t.Errorf("URL = %q; want to contain %q", out.URL, objectKey)
	}
	// ValidUntil should be in the future but within the next 3h
	if time.Until(out.ValidUntil) <= 0 || time.Until(out.ValidUntil) > 3*time.Hour {
		t.Errorf("ValidUntil = %v; want within next 3h", out.ValidUntil)
	}
	if out.Metadata.MimeType != "application/pdf" {
		t.Errorf("MimeType = %q; want %q", out.Metadata.MimeType, "application/pdf")
	}
	if out.Metadata.SizeBytes != int64(len(content)) {
		t.Errorf("SizeBytes = %d; want %d", out.Metadata.SizeBytes, len(content))
	}
	if out.Metadata.PageCount != 4 {
		t.Errorf("PageCount = %d; want %d", out.Metadata.PageCount, 4)
	}
	// documents should have no variants
	if len(out.Variants) != 0 {
		t.Errorf("Variants length = %d; want 0 for documents", len(out.Variants))
	}
}

func TestGetMediaIntegration_SuccessImageWithVariants(t *testing.T) {
	ctx := context.Background()

	testDB, err := testutil.SetupTestDB()
	if err != nil {
		t.Fatalf("setup DB: %v", err)
	}
	defer testDB.Cleanup()
	if err := migration.MigrateUp(testDB.DB); err != nil {
		t.Fatalf("migrate DB: %v", err)
	}

	bCleanup, err := testutil.SetupTestBuckets(GlobalStrg)
	if err != nil {
		t.Fatalf("setup buckets: %v", err)
	}
	defer bCleanup()

	mediaRepo := mariadb.NewMediaRepository(testDB.DB)
	ca := cache.NewNoop()
	svc := mediaSvc.NewMediaGetter(mediaRepo, ca, GlobalStrg)

	id := db.UUID(uuid.MustParse("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"))
	objectKey := id.String() + ".png"
	bucket := "images"

	// Upload original image
	width, height := 800, 600
	origContent := testutil.GeneratePNG(t, width, height)
	meta := model.Metadata{Width: width, Height: height}
	sizeBytes := int64(len(origContent))
	if err := GlobalStrg.SaveFile(ctx, bucket, objectKey, bytes.NewReader(origContent), sizeBytes, map[string]string{
		"Content-Type": "image/png",
	}); err != nil {
		t.Fatalf("upload original: %v", err)
	}

	variants := []model.Variant{
		{Width: 150, Height: height * 150 / width, ObjectKey: "variants/" + id.String() + "_150.png"},
		{Width: 300, Height: height * 300 / width, ObjectKey: "variants/" + id.String() + "_300.png"},
		{Width: 600, Height: height * 600 / width, ObjectKey: "variants/" + id.String() + "_600.png"},
	}
	// Upload variants
	for i := range variants {
		v := &variants[i]
		content := testutil.GeneratePNG(t, v.Width, v.Height)
		v.SizeBytes = int64(len(content))
		if err := GlobalStrg.SaveFile(ctx, bucket, v.ObjectKey, bytes.NewReader(content), v.SizeBytes, map[string]string{
			"Content-Type": "image/png",
		}); err != nil {
			t.Fatalf("upload variant %d: %v", v.Width, err)
		}
	}

	m := &model.Media{
		ID:        id,
		ObjectKey: objectKey,
		Bucket:    bucket,
		Status:    model.MediaStatusCompleted,
		Metadata:  meta,
		SizeBytes: &sizeBytes,
		MimeType:  ptrString("image/png"),
		Optimised: true,
		Variants:  variants,
	}
	if err := mediaRepo.Create(ctx, m); err != nil {
		t.Fatalf("insert media: %v", err)
	}

	out, err := svc.GetMedia(ctx, mediaSvc.GetMediaInput{ID: id})
	if err != nil {
		t.Fatalf("GetMedia returned error: %v", err)
	}

	// Assert original URL
	if !strings.Contains(out.URL, objectKey) {
		t.Errorf("original URL = %q; want contain %q", out.URL, objectKey)
	}
	// valid_until within 3h
	if d := time.Until(out.ValidUntil); d <= 0 || d > 3*time.Hour {
		t.Errorf("ValidUntil = %v; want within next 3h", out.ValidUntil)
	}

	// Assert Metadata
	if out.Metadata.MimeType != "image/png" {
		t.Errorf("MimeType = %q; want image/png", out.Metadata.MimeType)
	}
	if out.Metadata.SizeBytes != sizeBytes {
		t.Errorf("SizeBytes = %d; want %d", out.Metadata.SizeBytes, sizeBytes)
	}
	if out.Metadata.Width != meta.Width || out.Metadata.Height != meta.Height {
		t.Errorf("Width×Height = %dx%d; want %dx%d",
			out.Metadata.Width, out.Metadata.Height, meta.Width, meta.Height)
	}

	// Assert Variants
	if len(out.Variants) != len(m.Variants) {
		t.Fatalf("got %d variants; want %d", len(out.Variants), len(m.Variants))
	}
	// map widths→Variant
	byWidth := make(map[int]model.VariantOutput, len(out.Variants))
	for _, vo := range out.Variants {
		byWidth[vo.Width] = vo
	}
	for _, exp := range m.Variants {
		vo, ok := byWidth[exp.Width]
		if !ok {
			t.Errorf("missing variant for width %d", exp.Width)
			continue
		}
		// URL must contain the ObjectKey
		if !strings.Contains(vo.URL, exp.ObjectKey) {
			t.Errorf("variant URL = %q; want contain %q", vo.URL, exp.ObjectKey)
		}
		// metadata fields match
		if vo.Height != exp.Height {
			t.Errorf("variant %d height = %d; want %d", exp.Width, vo.Height, exp.Height)
		}
		if vo.SizeBytes != exp.SizeBytes {
			t.Errorf("variant %d size_bytes = %d; want %d", exp.Width, vo.SizeBytes, exp.SizeBytes)
		}
	}
}

func TestGetMediaIntegration_ErrorNotFound(t *testing.T) {
	testDB, _ := testutil.SetupTestDB()
	defer testDB.Cleanup()
	if err := migration.MigrateUp(testDB.DB); err != nil {
		t.Fatalf("migrate DB: %v", err)
	}

	bCleanup, err := testutil.SetupTestBuckets(GlobalStrg)
	if err != nil {
		t.Fatalf("setup buckets: %v", err)
	}
	defer bCleanup()

	repo := mariadb.NewMediaRepository(testDB.DB)
	ca := cache.NewNoop()
	svc := mediaSvc.NewMediaGetter(repo, ca, GlobalStrg)

	r := chi.NewRouter()
	r.With(api.WithID()).Get("/medias/{id}", api.GetMediaHandler(svc))

	// Make request for a non-existent UUID
	id := uuid.NewString()
	req := httptest.NewRequest(http.MethodGet, "/medias/"+id, nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	res := rec.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusNotFound {
		t.Errorf("status = %d; want %d", res.StatusCode, http.StatusNotFound)
	}
	if ct := res.Header.Get("Cache-Control"); ct != "no-store, max-age=0, must-revalidate" {
		t.Errorf("Cache-Control = %q; want no-store...", ct)
	}

	var resp errorResponse
	if err := json.NewDecoder(res.Body).Decode(&resp); err != nil {
		t.Fatalf("decode JSON: %v", err)
	}
	if !strings.Contains(resp.Error, "Media not found") {
		t.Errorf("error = %q; want contain %q", resp.Error, "Media not found")
	}
}

func TestGetMediaIntegration_ErrorInvalidID(t *testing.T) {
	// no DB or bucket setup needed, middleware will reject
	repo := mariadb.NewMediaRepository(nil)
	svc := mediaSvc.NewMediaGetter(repo, nil, nil)

	r := chi.NewRouter()
	r.With(api.WithID()).Get("/medias/{id}", api.GetMediaHandler(svc))

	// Invalid UUID
	req := httptest.NewRequest(http.MethodGet, "/medias/not-a-uuid", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	res := rec.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d; want %d", res.StatusCode, http.StatusBadRequest)
	}

	var resp errorResponse
	if err := json.NewDecoder(res.Body).Decode(&resp); err != nil {
		t.Fatalf("decode JSON: %v", err)
	}
	want := `ID "not-a-uuid" is not a valid UUID`
	if resp.Error != want {
		t.Errorf("error = %q; want %q", resp.Error, want)
	}
	if cc := res.Header.Get("Cache-Control"); cc != "no-store, max-age=0, must-revalidate" {
		t.Errorf("Cache-Control = %q; want no-store...", cc)
	}
}
