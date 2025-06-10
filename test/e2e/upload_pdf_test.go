package e2e

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/fhuszti/medias-ms-go/internal/cache"
	"github.com/fhuszti/medias-ms-go/internal/db"
	"github.com/fhuszti/medias-ms-go/internal/handler/api"
	"github.com/fhuszti/medias-ms-go/internal/migration"
	"github.com/fhuszti/medias-ms-go/internal/repository/mariadb"
	"github.com/fhuszti/medias-ms-go/internal/task"
	mediaSvc "github.com/fhuszti/medias-ms-go/internal/usecase/media"
	"github.com/fhuszti/medias-ms-go/test/testutil"
	"github.com/go-chi/chi/v5"
)

func waitOptimised(t *testing.T, url, id string, wantsVariants bool) mediaSvc.GetMediaOutput {
	t.Helper()

	var getOut mediaSvc.GetMediaOutput
	deadline := time.Now().Add(10 * time.Second)
	for {
		resp3, err := http.Get(url + "/medias/" + id)
		if err != nil {
			t.Fatalf("GET media error: %v", err)
		}
		if resp3.StatusCode != http.StatusOK {
			t.Fatalf("status GET media = %d; want %d", resp3.StatusCode, http.StatusOK)
		}
		if err := json.NewDecoder(resp3.Body).Decode(&getOut); err != nil {
			resp3.Body.Close()
			t.Fatalf("decode GET media JSON: %v", err)
		}
		resp3.Body.Close()
		if getOut.Optimised && (!wantsVariants || len(getOut.Variants) > 0) {
			return getOut
		}
		if time.Now().After(deadline) {
			t.Fatalf("timeout waiting for optimisation of %s", id)
		}
		time.Sleep(500 * time.Millisecond)
	}
}

func setupServer(t *testing.T) *httptest.Server {
	t.Helper()

	// Setup database
	testDB, err := testutil.SetupTestDB()
	if err != nil {
		t.Fatalf("setup DB: %v", err)
	}
	t.Cleanup(func() { _ = testDB.Cleanup() })
	dbConn := testDB.DB
	if err := migration.MigrateUp(dbConn); err != nil {
		t.Fatalf("could not run migrations: %v", err)
	}

	// Setup buckets
	bCleanup, err := testutil.SetupTestBuckets(GlobalStrg)
	if err != nil {
		t.Fatalf("setup buckets: %v", err)
	}
	t.Cleanup(func() { _ = bCleanup() })

	// Initialize repo and services
	repo := mariadb.NewMediaRepository(dbConn)
	uploadLinkSvc := mediaSvc.NewUploadLinkGenerator(repo, GlobalStrg, db.NewUUID)
	finaliserSvc := mediaSvc.NewUploadFinaliser(repo, GlobalStrg, task.NewDispatcher(RedisAddr, ""))
	workerStop := testutil.StartWorker(&db.Database{dbConn}, GlobalStrg, RedisAddr)
	t.Cleanup(workerStop)
	ca := cache.NewNoop()
	getterSvc := mediaSvc.NewMediaGetter(repo, ca, GlobalStrg)

	// Setup HTTP handlers
	r := chi.NewRouter()
	r.Post("/medias/generate_upload_link", api.GenerateUploadLinkHandler(uploadLinkSvc))
	r.With(api.WithDestBucket([]string{"staging", "images", "docs"})).
		Post("/medias/finalise_upload/{destBucket}", api.FinaliseUploadHandler(finaliserSvc))
	r.With(api.WithID()).
		Get("/medias/{id}", api.GetMediaHandler(getterSvc))

	ts := httptest.NewServer(r)
	t.Cleanup(ts.Close)

	return ts
}

func TestUploadImageE2E(t *testing.T) {
	ts := setupServer(t)

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
	raw := testutil.GeneratePNG(t, 200, 150)
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
	if resp2.StatusCode != http.StatusNoContent {
		t.Fatalf("status finalise_upload = %d; want %d", resp2.StatusCode, http.StatusNoContent)
	}

	// ---- Step 4: Poll GET media details until optimised ----
	getOut := waitOptimised(t, ts.URL, out1.ID, true)

	// Validate output
	if time.Until(getOut.ValidUntil) <= 0 {
		t.Errorf("ValidUntil = %v; want a time in the future", getOut.ValidUntil)
	}
	if !getOut.Optimised {
		t.Errorf("Optimised = %v; want true", getOut.Optimised)
	}
	if getOut.URL == "" {
		t.Error("URL is empty; want a presigned download URL")
	} else if _, err := url.Parse(getOut.URL); err != nil {
		t.Errorf("URL = %q; not a valid URL: %v", getOut.URL, err)
	}
	if getOut.Metadata.SizeBytes == 0 {
		t.Error("Metadata.SizeBytes = 0; want non-zero")
	}
	if getOut.Metadata.MimeType != "image/webp" {
		t.Errorf("Metadata.MimeType = %q; want image/webp", getOut.Metadata.MimeType)
	}
	if getOut.Metadata.Width != 200 {
		t.Errorf("Metadata.Width = %d; want 200", getOut.Metadata.Width)
	}
	if getOut.Metadata.Height != 150 {
		t.Errorf("Metadata.Height = %d; want 150", getOut.Metadata.Height)
	}
	if len(getOut.Variants) != 2 {
		t.Fatalf("Variants length = %d; want 2", len(getOut.Variants))
	}
	var found50, found300 bool
	for _, v := range getOut.Variants {
		if v.Width == 50 {
			if v.Height != 37 {
				t.Errorf("variant 50 height = %d; want 37", v.Height)
			}
			found50 = true
		} else if v.Width == 200 {
			if v.Height != 150 {
				t.Errorf("variant 300 dims = %dx%d; want 200x150", v.Width, v.Height)
			}
			found300 = true
		}
		if v.URL == "" {
			t.Error("variant URL empty; want non-empty presigned URL")
		} else if _, err := url.Parse(v.URL); err != nil {
			t.Errorf("variant URL = %q; invalid: %v", v.URL, err)
		}
		if v.SizeBytes == 0 {
			t.Errorf("variant size = %d; want >0", v.SizeBytes)
		}
	}
	if !found50 || !found300 {
		t.Error("expected variants for widths 50 and 300")
	}
}

func TestUploadMarkdownE2E(t *testing.T) {
	ts := setupServer(t)

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
	if resp2.StatusCode != http.StatusNoContent {
		t.Fatalf("status finalise_upload = %d; want %d", resp2.StatusCode, http.StatusNoContent)
	}

	// ---- Step 4: Poll GET media details until optimised ----
	getOut := waitOptimised(t, ts.URL, out1.ID, false)

	// Validate output
	if time.Until(getOut.ValidUntil) <= 0 {
		t.Errorf("ValidUntil = %v; want a time in the future", getOut.ValidUntil)
	}
	if !getOut.Optimised {
		t.Errorf("Optimised = %v; want true", getOut.Optimised)
	}
	if getOut.URL == "" {
		t.Error("URL is empty; want a presigned download URL")
	} else if _, err := url.Parse(getOut.URL); err != nil {
		t.Errorf("URL = %q; not a valid URL: %v", getOut.URL, err)
	}
	if getOut.Metadata.SizeBytes == 0 {
		t.Error("Metadata.SizeBytes = 0; want non-zero")
	}
	if getOut.Metadata.MimeType != "text/markdown" {
		t.Errorf("Metadata.MimeType = %q; want text/markdown", getOut.Metadata.MimeType)
	}
	if getOut.Metadata.WordCount != 23 {
		t.Errorf("Metadata.WordCount = %d; want 23", getOut.Metadata.WordCount)
	}
	if getOut.Metadata.HeadingCount != 3 {
		t.Errorf("Metadata.HeadingCount = %d; want 3", getOut.Metadata.HeadingCount)
	}
	if getOut.Metadata.LinkCount != 2 {
		t.Errorf("Metadata.LinkCount = %d; want 2", getOut.Metadata.LinkCount)
	}
	if len(getOut.Variants) != 0 {
		t.Errorf("Variants = %v; want empty slice", getOut.Variants)
	}
}

func TestUploadPDFE2E(t *testing.T) {
	ts := setupServer(t)

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
	if resp2.StatusCode != http.StatusNoContent {
		t.Fatalf("status finalise_upload = %d; want %d", resp2.StatusCode, http.StatusNoContent)
	}

	// ---- Step 4: Poll GET media details until optimised ----
	getOut := waitOptimised(t, ts.URL, out1.ID, false)

	// Validate output
	if time.Until(getOut.ValidUntil) <= 0 {
		t.Errorf("ValidUntil = %v; want a time in the future", getOut.ValidUntil)
	}
	if !getOut.Optimised {
		t.Errorf("Optimised = %v; want true", getOut.Optimised)
	}
	if getOut.URL == "" {
		t.Error("URL is empty; want a presigned download URL")
	} else if _, err := url.Parse(getOut.URL); err != nil {
		t.Errorf("URL = %q; not a valid URL: %v", getOut.URL, err)
	}
	if getOut.Metadata.SizeBytes == 0 {
		t.Error("Metadata.SizeBytes = 0; want non-zero")
	}
	if getOut.Metadata.MimeType != "application/pdf" {
		t.Errorf("Metadata.MimeType = %q; want application/pdf", getOut.Metadata.MimeType)
	}
	if getOut.Metadata.PageCount != 4 {
		t.Errorf("Metadata.PageCount = %d; want 4", getOut.Metadata.PageCount)
	}
	if len(getOut.Variants) != 0 {
		t.Errorf("Variants = %v; want empty slice", getOut.Variants)
	}
}
