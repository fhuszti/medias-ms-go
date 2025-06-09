package e2e

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/fhuszti/medias-ms-go/internal/cache"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/fhuszti/medias-ms-go/internal/db"
	"github.com/fhuszti/medias-ms-go/internal/handler/api"
	"github.com/fhuszti/medias-ms-go/internal/migration"
	"github.com/fhuszti/medias-ms-go/internal/model"
	"github.com/fhuszti/medias-ms-go/internal/repository/mariadb"
	"github.com/fhuszti/medias-ms-go/internal/task"
	mediaSvc "github.com/fhuszti/medias-ms-go/internal/usecase/media"
	"github.com/fhuszti/medias-ms-go/test/testutil"
	"github.com/go-chi/chi/v5"
)

func TestUploadImageE2E(t *testing.T) {
	// Setup database
	testDB, err := testutil.SetupTestDB()
	if err != nil {
		t.Fatalf("setup DB: %v", err)
	}
	defer testDB.Cleanup()
	dbConn := testDB.DB
	if err := migration.MigrateUp(dbConn); err != nil {
		t.Fatalf("could not run migrations: %v", err)
	}

	// Setup buckets
	bCleanup, err := testutil.SetupTestBuckets(GlobalStrg)
	if err != nil {
		t.Fatalf("setup buckets: %v", err)
	}
	defer bCleanup()

	// Initialize repo and services
	repo := mariadb.NewMediaRepository(dbConn)
	uploadLinkSvc := mediaSvc.NewUploadLinkGenerator(repo, GlobalStrg, db.NewUUID)
	finaliserSvc := mediaSvc.NewUploadFinaliser(repo, GlobalStrg, task.NewNoopDispatcher())
	ca := cache.NewNoop()
	getterSvc := mediaSvc.NewMediaGetter(repo, ca, GlobalStrg)

	// Setup HTTP handlers
	r := chi.NewRouter()
	r.Post("/medias/generate_upload_link", api.GenerateUploadLinkHandler(uploadLinkSvc))
	r.With(api.WithDestBucket([]string{"staging", "images"})).
		Post("/medias/finalise_upload/{destBucket}", api.FinaliseUploadHandler(finaliserSvc))
	r.With(api.WithID()).
		Get("/medias/{id}", api.GetMediaHandler(getterSvc))

	ts := httptest.NewServer(r)
	defer ts.Close()

	// ---- Step 1: Generate upload link ----
	genReq := `{"name":"sample.png"}`
	resp1, err := http.Post(ts.URL+"/medias/generate_upload_link", "application/json", strings.NewReader(genReq))
	if err != nil {
		t.Fatalf("POST generate_upload_link error: %v", err)
	}
	defer resp1.Body.Close()
	if resp1.StatusCode != http.StatusCreated {
		t.Fatalf("status generate_upload_link = %d; want %d", resp1.StatusCode, http.StatusCreated)
	}
	var out1 struct {
		ID  string `json:"id"`
		URL string `json:"url"`
	}
	if err := json.NewDecoder(resp1.Body).Decode(&out1); err != nil {
		t.Fatalf("decode generate_upload_link JSON: %v", err)
	}
	if out1.ID == "" {
		t.Fatal("expected non-empty ID from generate_upload_link")
	}
	if out1.URL == "" {
		t.Fatal("expected non-empty URL from generate_upload_link")
	}

	// ---- Step 2: PUT a PNG file to the presigned URL ----
	raw := testutil.GeneratePNG(t, 800, 600)
	reqPut, err := http.NewRequest(http.MethodPut, out1.URL, bytes.NewReader(raw))
	if err != nil {
		t.Fatalf("new PUT request error: %v", err)
	}
	reqPut.Header.Set("Content-Type", "image/png")
	putResp, err := http.DefaultClient.Do(reqPut)
	if err != nil {
		t.Fatalf("PUT to presigned URL error: %v", err)
	}
	putResp.Body.Close()
	if putResp.StatusCode < 200 || putResp.StatusCode >= 300 {
		t.Fatalf("presigned PUT status = %d; want 2xx", putResp.StatusCode)
	}

	// ---- Step 3: Finalise upload ----
	finReq := fmt.Sprintf(`{"id":"%s"}`, out1.ID)
	resp2, err := http.Post(ts.URL+"/medias/finalise_upload/images", "application/json", strings.NewReader(finReq))
	if err != nil {
		t.Fatalf("POST finalise_upload error: %v", err)
	}
	defer resp2.Body.Close()
	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("status finalise_upload = %d; want %d", resp2.StatusCode, http.StatusOK)
	}
	var mediaOut model.Media
	if err := json.NewDecoder(resp2.Body).Decode(&mediaOut); err != nil {
		t.Fatalf("decode finalise_upload JSON: %v", err)
	}

	// Basic assertions on finalised output
	if mediaOut.ID.String() != out1.ID {
		t.Errorf("finalise ID = %q; want %q", mediaOut.ID, out1.ID)
	}
	if mediaOut.Bucket != "images" {
		t.Errorf("finalise bucket = %q; want images", mediaOut.Bucket)
	}
	if mediaOut.ObjectKey != out1.ID+".png" {
		t.Errorf("finalise ObjectKey = %q; want %q", mediaOut.ObjectKey, out1.ID+".png")
	}
	if mediaOut.Status != model.MediaStatusCompleted {
		t.Errorf("finalise status = %q; want %q", mediaOut.Status, model.MediaStatusCompleted)
	}
	if mediaOut.SizeBytes == nil || *mediaOut.SizeBytes != int64(len(raw)) {
		t.Errorf("finalise SizeBytes = %v; want %d", mediaOut.SizeBytes, len(raw))
	}
	if mediaOut.MimeType == nil || *mediaOut.MimeType != "image/png" {
		t.Errorf("finalise MimeType = %q; want image/png", *mediaOut.MimeType)
	}
	if mediaOut.Metadata.Width != 800 {
		t.Errorf("finalise Width = %d; want 800", mediaOut.Metadata.Width)
	}
	if mediaOut.Metadata.Height != 600 {
		t.Errorf("finalise Height = %d; want 600", mediaOut.Metadata.Height)
	}

	// ---- Step 4: GET media details ----
	resp3, err := http.Get(ts.URL + "/medias/" + out1.ID)
	if err != nil {
		t.Fatalf("GET media error: %v", err)
	}
	defer resp3.Body.Close()
	if resp3.StatusCode != http.StatusOK {
		t.Fatalf("status GET media = %d; want %d", resp3.StatusCode, http.StatusOK)
	}
	var getOut mediaSvc.GetMediaOutput
	if err := json.NewDecoder(resp3.Body).Decode(&getOut); err != nil {
		t.Fatalf("decode GET media JSON: %v", err)
	}
	// Validate output
	if time.Until(getOut.ValidUntil) <= 0 {
		t.Errorf("ValidUntil = %v; want a time in the future", getOut.ValidUntil)
	}
	if getOut.Optimised {
		t.Errorf("Optimised = %v; want false", getOut.Optimised)
	}
	if getOut.URL == "" {
		t.Error("URL is empty; want a presigned download URL")
	} else if _, err := url.Parse(getOut.URL); err != nil {
		t.Errorf("URL = %q; not a valid URL: %v", getOut.URL, err)
	}
	if getOut.Metadata.SizeBytes != *mediaOut.SizeBytes {
		t.Errorf("Metadata.SizeBytes = %d; want %d", getOut.Metadata.SizeBytes, *mediaOut.SizeBytes)
	}
	if getOut.Metadata.MimeType != *mediaOut.MimeType {
		t.Errorf("Metadata.MimeType = %q; want %q", getOut.Metadata.MimeType, *mediaOut.MimeType)
	}
	if getOut.Metadata.Width != mediaOut.Metadata.Width {
		t.Errorf("Metadata.Width = %d; want %d", getOut.Metadata.Width, mediaOut.Metadata.Width)
	}
	if getOut.Metadata.Height != mediaOut.Metadata.Height {
		t.Errorf("Metadata.Height = %d; want %d", getOut.Metadata.Height, mediaOut.Metadata.Height)
	}
	if len(getOut.Variants) != 0 {
		t.Errorf("Variants = %v; want empty slice", getOut.Variants)
	}
}

func TestUploadMarkdownE2E(t *testing.T) {
	// Setup database
	testDB, err := testutil.SetupTestDB()
	if err != nil {
		t.Fatalf("setup DB: %v", err)
	}
	defer testDB.Cleanup()
	dbConn := testDB.DB
	if err := migration.MigrateUp(dbConn); err != nil {
		t.Fatalf("could not run migrations: %v", err)
	}

	// Setup buckets
	bCleanup, err := testutil.SetupTestBuckets(GlobalStrg)
	if err != nil {
		t.Fatalf("setup buckets: %v", err)
	}
	defer bCleanup()

	// Initialize repo and services
	repo := mariadb.NewMediaRepository(dbConn)
	uploadLinkSvc := mediaSvc.NewUploadLinkGenerator(repo, GlobalStrg, db.NewUUID)
	finaliserSvc := mediaSvc.NewUploadFinaliser(repo, GlobalStrg, task.NewNoopDispatcher())
	ca := cache.NewNoop()
	getterSvc := mediaSvc.NewMediaGetter(repo, ca, GlobalStrg)

	// Setup HTTP handlers
	r := chi.NewRouter()
	r.Post("/medias/generate_upload_link", api.GenerateUploadLinkHandler(uploadLinkSvc))
	r.With(api.WithDestBucket([]string{"staging", "docs"})).
		Post("/medias/finalise_upload/{destBucket}", api.FinaliseUploadHandler(finaliserSvc))
	r.With(api.WithID()).
		Get("/medias/{id}", api.GetMediaHandler(getterSvc))

	ts := httptest.NewServer(r)
	defer ts.Close()

	// ---- Step 1: Generate upload link ----
	genReq := `{"name":"sample.md"}`
	resp1, err := http.Post(ts.URL+"/medias/generate_upload_link", "application/json", strings.NewReader(genReq))
	if err != nil {
		t.Fatalf("POST generate_upload_link error: %v", err)
	}
	defer resp1.Body.Close()
	if resp1.StatusCode != http.StatusCreated {
		t.Fatalf("status generate_upload_link = %d; want %d", resp1.StatusCode, http.StatusCreated)
	}
	var out1 struct {
		ID  string `json:"id"`
		URL string `json:"url"`
	}
	if err := json.NewDecoder(resp1.Body).Decode(&out1); err != nil {
		t.Fatalf("decode generate_upload_link JSON: %v", err)
	}
	if out1.ID == "" {
		t.Fatal("expected non-empty ID from generate_upload_link")
	}
	if out1.URL == "" {
		t.Fatal("expected non-empty URL from generate_upload_link")
	}

	// ---- Step 2: PUT a Markdown file to the presigned URL ----
	raw := testutil.GenerateMarkdown()
	reqPut, err := http.NewRequest(http.MethodPut, out1.URL, bytes.NewReader(raw))
	if err != nil {
		t.Fatalf("new PUT request error: %v", err)
	}
	reqPut.Header.Set("Content-Type", "text/markdown")
	putResp, err := http.DefaultClient.Do(reqPut)
	if err != nil {
		t.Fatalf("PUT to presigned URL error: %v", err)
	}
	putResp.Body.Close()
	if putResp.StatusCode < 200 || putResp.StatusCode >= 300 {
		t.Fatalf("presigned PUT status = %d; want 2xx", putResp.StatusCode)
	}

	// ---- Step 3: Finalise upload ----
	finReq := fmt.Sprintf(`{"id":"%s"}`, out1.ID)
	resp2, err := http.Post(ts.URL+"/medias/finalise_upload/docs", "application/json", strings.NewReader(finReq))
	if err != nil {
		t.Fatalf("POST finalise_upload error: %v", err)
	}
	defer resp2.Body.Close()
	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("status finalise_upload = %d; want %d", resp2.StatusCode, http.StatusOK)
	}
	var mediaOut model.Media
	if err := json.NewDecoder(resp2.Body).Decode(&mediaOut); err != nil {
		t.Fatalf("decode finalise_upload JSON: %v", err)
	}

	// Basic assertions on finalise output
	if mediaOut.ID.String() != out1.ID {
		t.Errorf("finalise ID = %q; want %q", mediaOut.ID, out1.ID)
	}
	if mediaOut.Bucket != "docs" {
		t.Errorf("finalise bucket = %q; want docs", mediaOut.Bucket)
	}
	if mediaOut.ObjectKey != out1.ID+".md" {
		t.Errorf("finalise ObjectKey = %q; want %q", mediaOut.ObjectKey, out1.ID+".md")
	}
	if mediaOut.Status != model.MediaStatusCompleted {
		t.Errorf("finalise status = %q; want %q", mediaOut.Status, model.MediaStatusCompleted)
	}
	if mediaOut.SizeBytes == nil || *mediaOut.SizeBytes != int64(len(raw)) {
		t.Errorf("finalise SizeBytes = %v; want %d", mediaOut.SizeBytes, len(raw))
	}
	if mediaOut.MimeType == nil || *mediaOut.MimeType != "text/markdown" {
		t.Errorf("finalise MimeType = %q; want text/markdown", *mediaOut.MimeType)
	}
	if mediaOut.Metadata.WordCount != 23 {
		t.Errorf("finalise WordCount = %d; want 23", mediaOut.Metadata.WordCount)
	}
	if mediaOut.Metadata.HeadingCount != 3 {
		t.Errorf("finalise HeadingCount = %d; want 3", mediaOut.Metadata.HeadingCount)
	}
	if mediaOut.Metadata.LinkCount != 2 {
		t.Errorf("finalise LinkCount = %d; want 2", mediaOut.Metadata.LinkCount)
	}

	// ---- Step 4: GET media details ----
	resp3, err := http.Get(ts.URL + "/medias/" + out1.ID)
	if err != nil {
		t.Fatalf("GET media error: %v", err)
	}
	defer resp3.Body.Close()
	if resp3.StatusCode != http.StatusOK {
		t.Fatalf("status GET media = %d; want %d", resp3.StatusCode, http.StatusOK)
	}
	var getOut mediaSvc.GetMediaOutput
	if err := json.NewDecoder(resp3.Body).Decode(&getOut); err != nil {
		t.Fatalf("decode GET media JSON: %v", err)
	}
	// Validate output
	if time.Until(getOut.ValidUntil) <= 0 {
		t.Errorf("ValidUntil = %v; want a time in the future", getOut.ValidUntil)
	}
	if getOut.Optimised {
		t.Errorf("Optimised = %v; want false", getOut.Optimised)
	}
	if getOut.URL == "" {
		t.Error("URL is empty; want a presigned download URL")
	} else if _, err := url.Parse(getOut.URL); err != nil {
		t.Errorf("URL = %q; not a valid URL: %v", getOut.URL, err)
	}
	if getOut.Metadata.SizeBytes != *mediaOut.SizeBytes {
		t.Errorf("Metadata.SizeBytes = %d; want %d", getOut.Metadata.SizeBytes, *mediaOut.SizeBytes)
	}
	if getOut.Metadata.MimeType != *mediaOut.MimeType {
		t.Errorf("Metadata.MimeType = %q; want %q", getOut.Metadata.MimeType, *mediaOut.MimeType)
	}
	if getOut.Metadata.WordCount != mediaOut.Metadata.WordCount {
		t.Errorf("Metadata.WordCount = %d; want %d", getOut.Metadata.WordCount, mediaOut.Metadata.WordCount)
	}
	if getOut.Metadata.HeadingCount != mediaOut.Metadata.HeadingCount {
		t.Errorf("Metadata.HeadingCount = %d; want %d", getOut.Metadata.HeadingCount, mediaOut.Metadata.HeadingCount)
	}
	if getOut.Metadata.LinkCount != mediaOut.Metadata.LinkCount {
		t.Errorf("Metadata.LinkCount = %d; want %d", getOut.Metadata.LinkCount, mediaOut.Metadata.LinkCount)
	}
	if len(getOut.Variants) != 0 {
		t.Errorf("Variants = %v; want empty slice", getOut.Variants)
	}
}

func TestUploadPDFE2E(t *testing.T) {
	// Setup database
	testDB, err := testutil.SetupTestDB()
	if err != nil {
		t.Fatalf("setup DB: %v", err)
	}
	defer testDB.Cleanup()
	dbConn := testDB.DB
	if err := migration.MigrateUp(dbConn); err != nil {
		t.Fatalf("could not run migrations: %v", err)
	}

	// Setup buckets
	bCleanup, err := testutil.SetupTestBuckets(GlobalStrg)
	if err != nil {
		t.Fatalf("setup buckets: %v", err)
	}
	defer bCleanup()

	// Initialize repo and services
	repo := mariadb.NewMediaRepository(dbConn)
	uploadLinkSvc := mediaSvc.NewUploadLinkGenerator(repo, GlobalStrg, db.NewUUID)
	finaliserSvc := mediaSvc.NewUploadFinaliser(repo, GlobalStrg, task.NewNoopDispatcher())
	ca := cache.NewNoop()
	getterSvc := mediaSvc.NewMediaGetter(repo, ca, GlobalStrg)

	// Setup HTTP handlers
	r := chi.NewRouter()
	r.Post("/medias/generate_upload_link", api.GenerateUploadLinkHandler(uploadLinkSvc))
	r.With(api.WithDestBucket([]string{"staging", "docs"})).
		Post("/medias/finalise_upload/{destBucket}", api.FinaliseUploadHandler(finaliserSvc))
	r.With(api.WithID()).
		Get("/medias/{id}", api.GetMediaHandler(getterSvc))

	ts := httptest.NewServer(r)
	defer ts.Close()

	// ---- Step 1: Generate upload link ----
	genReq := `{"name":"sample.pdf"}`
	resp1, err := http.Post(ts.URL+"/medias/generate_upload_link", "application/json", strings.NewReader(genReq))
	if err != nil {
		t.Fatalf("POST generate_upload_link error: %v", err)
	}
	defer resp1.Body.Close()
	if resp1.StatusCode != http.StatusCreated {
		t.Fatalf("status generate_upload_link = %d; want %d", resp1.StatusCode, http.StatusCreated)
	}
	var out1 struct {
		ID  string `json:"id"`
		URL string `json:"url"`
	}
	if err := json.NewDecoder(resp1.Body).Decode(&out1); err != nil {
		t.Fatalf("decode generate_upload_link JSON: %v", err)
	}
	if out1.ID == "" {
		t.Fatal("expected non-empty ID from generate_upload_link")
	}
	if out1.URL == "" {
		t.Fatal("expected non-empty URL from generate_upload_link")
	}

	// ---- Step 2: PUT PDF to presigned URL ----
	raw := testutil.LoadPDF(t)
	reqPut, err := http.NewRequest(http.MethodPut, out1.URL, bytes.NewReader(raw))
	if err != nil {
		t.Fatalf("new PUT request error: %v", err)
	}
	reqPut.Header.Set("Content-Type", "application/pdf")
	putResp, err := http.DefaultClient.Do(reqPut)
	if err != nil {
		t.Fatalf("PUT to presigned URL error: %v", err)
	}
	putResp.Body.Close()
	if putResp.StatusCode < 200 || putResp.StatusCode >= 300 {
		t.Fatalf("presigned PUT status = %d; want 2xx", putResp.StatusCode)
	}

	// ---- Step 3: Finalise upload ----
	finReq := fmt.Sprintf(`{"id":"%s"}`, out1.ID)
	resp2, err := http.Post(ts.URL+"/medias/finalise_upload/docs", "application/json", strings.NewReader(finReq))
	if err != nil {
		t.Fatalf("POST finalise_upload error: %v", err)
	}
	defer resp2.Body.Close()
	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("status finalise_upload = %d; want %d", resp2.StatusCode, http.StatusOK)
	}
	var mediaOut model.Media
	if err := json.NewDecoder(resp2.Body).Decode(&mediaOut); err != nil {
		t.Fatalf("decode finalise_upload JSON: %v", err)
	}

	// Basic assertions on finalise output
	if mediaOut.ID.String() != out1.ID {
		t.Errorf("finalise ID = %q; want %q", mediaOut.ID, out1.ID)
	}
	if mediaOut.Bucket != "docs" {
		t.Errorf("finalise bucket = %q; want docs", mediaOut.Bucket)
	}
	if mediaOut.ObjectKey != out1.ID+".pdf" {
		t.Errorf("finalise ObjectKey = %q; want %q", mediaOut.ObjectKey, out1.ID+".pdf")
	}
	if mediaOut.Status != model.MediaStatusCompleted {
		t.Errorf("finalise status = %q; want %q", mediaOut.Status, model.MediaStatusCompleted)
	}
	if mediaOut.SizeBytes == nil || *mediaOut.SizeBytes != int64(len(raw)) {
		t.Errorf("finalise SizeBytes = %v; want %d", mediaOut.SizeBytes, len(raw))
	}
	if mediaOut.MimeType == nil || *mediaOut.MimeType != "application/pdf" {
		t.Errorf("finalise MimeType = %q; want application/pdf", *mediaOut.MimeType)
	}
	if mediaOut.Metadata.PageCount != 4 {
		t.Errorf("finalise PageCount = %d; want 4", mediaOut.Metadata.PageCount)
	}

	// ---- Step 4: GET media details ----
	resp3, err := http.Get(ts.URL + "/medias/" + out1.ID)
	if err != nil {
		t.Fatalf("GET media error: %v", err)
	}
	defer resp3.Body.Close()
	if resp3.StatusCode != http.StatusOK {
		t.Fatalf("status GET media = %d; want %d", resp3.StatusCode, http.StatusOK)
	}
	var getOut mediaSvc.GetMediaOutput
	if err := json.NewDecoder(resp3.Body).Decode(&getOut); err != nil {
		t.Fatalf("decode GET media JSON: %v", err)
	}
	// Validate output
	if time.Until(getOut.ValidUntil) <= 0 {
		t.Errorf("ValidUntil = %v; want a time in the future", getOut.ValidUntil)
	}
	if getOut.Optimised {
		t.Errorf("Optimised = %v; want false", getOut.Optimised)
	}
	if getOut.URL == "" {
		t.Error("URL is empty; want a presigned download URL")
	} else if _, err := url.Parse(getOut.URL); err != nil {
		t.Errorf("URL = %q; not a valid URL: %v", getOut.URL, err)
	}
	if getOut.Metadata.SizeBytes != *mediaOut.SizeBytes {
		t.Errorf("Metadata.SizeBytes = %d; want %d", getOut.Metadata.SizeBytes, *mediaOut.SizeBytes)
	}
	if getOut.Metadata.MimeType != *mediaOut.MimeType {
		t.Errorf("Metadata.MimeType = %q; want %q", getOut.Metadata.MimeType, *mediaOut.MimeType)
	}
	if getOut.Metadata.PageCount != mediaOut.Metadata.PageCount {
		t.Errorf("Metadata.PageCount = %d; want %d", getOut.Metadata.PageCount, mediaOut.Metadata.PageCount)
	}
	if len(getOut.Variants) != 0 {
		t.Errorf("Variants = %v; want empty slice", getOut.Variants)
	}
}
