package e2e

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"testing"
	"time"

	mediaSvc "github.com/fhuszti/medias-ms-go/internal/usecase/media"
	"github.com/fhuszti/medias-ms-go/test/testutil"
)

type presignedResponse struct {
	ID  string `json:"id"`
	URL string `json:"url"`
}

func initiateUpload(t *testing.T, baseURL, name string) presignedResponse {
	t.Helper()

	genReq := fmt.Sprintf(`{"name":"%s"}`, name)
	resp, err := http.Post(baseURL+"/medias/initiate_upload", "application/json", strings.NewReader(genReq))
	if err != nil {
		t.Fatalf("POST initiate_upload error: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("status initiate_upload = %d; want %d", resp.StatusCode, http.StatusCreated)
	}
	var out presignedResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode initiate_upload JSON: %v", err)
	}
	if out.ID == "" {
		t.Fatal("expected non-empty ID from initiate_upload")
	}
	if out.URL == "" {
		t.Fatal("expected non-empty URL from initiate_upload")
	}
	return out
}

func initiateUpdate(t *testing.T, baseURL, id, name string) presignedResponse {
	t.Helper()

	genReq := fmt.Sprintf(`{"name":"%s"}`, name)
	resp, err := http.Post(baseURL+"/medias/"+id+"/initiate_update", "application/json", strings.NewReader(genReq))
	if err != nil {
		t.Fatalf("POST initiate_update error: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("status initiate_update = %d; want %d", resp.StatusCode, http.StatusCreated)
	}
	var out presignedResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode initiate_update JSON: %v", err)
	}
	if out.ID == "" {
		t.Fatal("expected non-empty ID from initiate_update")
	}
	if out.URL == "" {
		t.Fatal("expected non-empty URL from initiate_update")
	}
	return out
}

func putPresignedFile(t *testing.T, url, contentType string, data []byte) {
	t.Helper()

	reqPut, err := http.NewRequest(http.MethodPut, url, bytes.NewReader(data))
	if err != nil {
		t.Fatalf("new PUT request error: %v", err)
	}
	reqPut.Header.Set("Content-Type", contentType)
	putResp, err := http.DefaultClient.Do(reqPut)
	if err != nil {
		t.Fatalf("PUT to presigned URL error: %v", err)
	}
	putResp.Body.Close()
	if putResp.StatusCode < 200 || putResp.StatusCode >= 300 {
		t.Fatalf("presigned PUT status = %d; want 2xx", putResp.StatusCode)
	}
}

func finaliseUpload(t *testing.T, baseURL, id, destBucket string) {
	t.Helper()

	finReq := fmt.Sprintf(`{"dest_bucket":"%s"}`, destBucket)
	resp, err := http.Post(baseURL+"/medias/"+id+"/finalise_upload", "application/json", strings.NewReader(finReq))
	if err != nil {
		t.Fatalf("POST finalise_upload error: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("status finalise_upload = %d; want %d", resp.StatusCode, http.StatusNoContent)
	}
}

func finaliseUpdate(t *testing.T, baseURL, id, destBucket string) {
	t.Helper()

	finReq := fmt.Sprintf(`{"dest_bucket":"%s"}`, destBucket)
	resp, err := http.Post(baseURL+"/medias/"+id+"/finalise_update", "application/json", strings.NewReader(finReq))
	if err != nil {
		t.Fatalf("POST finalise_update error: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("status finalise_update = %d; want %d", resp.StatusCode, http.StatusNoContent)
	}
}

func markdownExpectations(content string) (words, headings, links int64) {
	words = int64(len(strings.Fields(content)))

	headingCount := 0
	for _, line := range strings.Split(content, "\n") {
		if strings.HasPrefix(line, "# ") || strings.HasPrefix(line, "##") {
			headingCount++
		}
	}
	headings = int64(headingCount)

	linkRe := regexp.MustCompile(`\[[^]]+]\([^)]+\)`)
	links = int64(len(linkRe.FindAllString(content, -1)))

	return words, headings, links
}

func TestUpdateImageE2E(t *testing.T) {
	ts := setupServer(t)

	// ---- Step 1: Upload initial image ----
	initOut := initiateUpload(t, ts.URL, "sample.png")
	putPresignedFile(t, initOut.URL, "image/png", testutil.GeneratePNG(t, 200, 150))
	finaliseUpload(t, ts.URL, initOut.ID, "images")
	_ = waitOptimised(t, ts.URL, initOut.ID, true)

	// ---- Step 2: Initiate update ----
	updOut := initiateUpdate(t, ts.URL, initOut.ID, "sample-v2.png")

	// ---- Step 3: PUT updated image ----
	putPresignedFile(t, updOut.URL, "image/png", testutil.GeneratePNG(t, 120, 80))

	// ---- Step 4: Finalise update ----
	finaliseUpdate(t, ts.URL, initOut.ID, "images")

	// ---- Step 5: Poll GET media details until optimised ----
	getOut := waitOptimised(t, ts.URL, initOut.ID, true)

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
	} else if !strings.Contains(getOut.URL, fmt.Sprintf("%s-v2", initOut.ID)) {
		t.Errorf("URL = %q; want to contain version v2", getOut.URL)
	}
	if getOut.Metadata.SizeBytes == 0 {
		t.Error("Metadata.SizeBytes = 0; want non-zero")
	}
	if getOut.Metadata.MimeType != "image/webp" {
		t.Errorf("Metadata.MimeType = %q; want image/webp", getOut.Metadata.MimeType)
	}
	if getOut.Metadata.Width != 120 {
		t.Errorf("Metadata.Width = %d; want 120", getOut.Metadata.Width)
	}
	if getOut.Metadata.Height != 80 {
		t.Errorf("Metadata.Height = %d; want 80", getOut.Metadata.Height)
	}
	if len(getOut.Variants) != 2 {
		t.Fatalf("Variants length = %d; want 2", len(getOut.Variants))
	}
	var found50, found300 bool
	key50 := fmt.Sprintf("variants/%s/%s-v2_50.webp", initOut.ID, initOut.ID)
	key300 := fmt.Sprintf("variants/%s/%s-v2_300.webp", initOut.ID, initOut.ID)
	for _, v := range getOut.Variants {
		if strings.Contains(v.URL, key50) {
			if v.Width != 50 || v.Height != 33 {
				t.Errorf("variant 50 dims = %dx%d; want 50x33", v.Width, v.Height)
			}
			found50 = true
		} else if strings.Contains(v.URL, key300) {
			if v.Width != 120 || v.Height != 80 {
				t.Errorf("variant 300 dims = %dx%d; want 120x80", v.Width, v.Height)
			}
			found300 = true
		}
		if strings.Contains(v.URL, fmt.Sprintf("%s-v1_", initOut.ID)) {
			t.Errorf("variant URL = %q; want v2 variants only", v.URL)
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
	if !found50 {
		t.Error("expected 50px variant")
	}
	if !found300 {
		t.Error("expected 300px variant")
	}

	// ---- Step 6: Delete the media ----
	reqDel, err := http.NewRequest(http.MethodDelete, ts.URL+"/medias/"+initOut.ID, nil)
	if err != nil {
		t.Fatalf("new DELETE request error: %v", err)
	}
	respDel, err := http.DefaultClient.Do(reqDel)
	if err != nil {
		t.Fatalf("DELETE media error: %v", err)
	}
	respDel.Body.Close()
	if respDel.StatusCode != http.StatusNoContent {
		t.Fatalf("status delete = %d; want %d", respDel.StatusCode, http.StatusNoContent)
	}

	// ---- Step 7: Confirm GET returns 404 ----
	respGet, err := http.Get(ts.URL + "/medias/" + initOut.ID)
	if err != nil {
		t.Fatalf("GET after delete error: %v", err)
	}
	defer respGet.Body.Close()
	if respGet.StatusCode != http.StatusNotFound {
		t.Fatalf("status after delete GET = %d; want %d", respGet.StatusCode, http.StatusNotFound)
	}
}

func TestUpdateMarkdownE2E(t *testing.T) {
	ts := setupServer(t)

	// ---- Step 1: Upload initial markdown ----
	initOut := initiateUpload(t, ts.URL, "sample.md")
	putPresignedFile(t, initOut.URL, "text/markdown", testutil.GenerateMarkdown())
	finaliseUpload(t, ts.URL, initOut.ID, "docs")
	_ = waitOptimised(t, ts.URL, initOut.ID, false)

	// ---- Step 2: Initiate update ----
	updOut := initiateUpdate(t, ts.URL, initOut.ID, "sample-v2.md")

	// ---- Step 3: PUT updated markdown ----
	updatedMarkdown := strings.Join([]string{
		"# Updated Title",
		"## Subtitle",
		"A short line with a [link](https://example.com).",
		"Another line with two words.",
		strings.Repeat("x", int(mediaSvc.MinFileSize)),
	}, "\n")
	wordCount, headingCount, linkCount := markdownExpectations(updatedMarkdown)
	putPresignedFile(t, updOut.URL, "text/markdown", []byte(updatedMarkdown))

	// ---- Step 4: Finalise update ----
	finaliseUpdate(t, ts.URL, initOut.ID, "docs")

	// ---- Step 5: Poll GET media details until optimised ----
	getOut := waitOptimised(t, ts.URL, initOut.ID, false)

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
	} else if !strings.Contains(getOut.URL, fmt.Sprintf("%s-v2", initOut.ID)) {
		t.Errorf("URL = %q; want to contain version v2", getOut.URL)
	}
	if getOut.Metadata.SizeBytes == 0 {
		t.Error("Metadata.SizeBytes = 0; want non-zero")
	}
	if getOut.Metadata.MimeType != "text/markdown" {
		t.Errorf("Metadata.MimeType = %q; want text/markdown", getOut.Metadata.MimeType)
	}
	if getOut.Metadata.WordCount != wordCount {
		t.Errorf("Metadata.WordCount = %d; want %d", getOut.Metadata.WordCount, wordCount)
	}
	if getOut.Metadata.HeadingCount != headingCount {
		t.Errorf("Metadata.HeadingCount = %d; want %d", getOut.Metadata.HeadingCount, headingCount)
	}
	if getOut.Metadata.LinkCount != linkCount {
		t.Errorf("Metadata.LinkCount = %d; want %d", getOut.Metadata.LinkCount, linkCount)
	}
	if len(getOut.Variants) != 0 {
		t.Errorf("Variants = %v; want empty slice", getOut.Variants)
	}

	// ---- Step 6: Delete the media ----
	reqDel, err := http.NewRequest(http.MethodDelete, ts.URL+"/medias/"+initOut.ID, nil)
	if err != nil {
		t.Fatalf("new DELETE request error: %v", err)
	}
	respDel, err := http.DefaultClient.Do(reqDel)
	if err != nil {
		t.Fatalf("DELETE media error: %v", err)
	}
	respDel.Body.Close()
	if respDel.StatusCode != http.StatusNoContent {
		t.Fatalf("status delete = %d; want %d", respDel.StatusCode, http.StatusNoContent)
	}

	// ---- Step 7: Confirm GET returns 404 ----
	respGet, err := http.Get(ts.URL + "/medias/" + initOut.ID)
	if err != nil {
		t.Fatalf("GET after delete error: %v", err)
	}
	defer respGet.Body.Close()
	if respGet.StatusCode != http.StatusNotFound {
		t.Fatalf("status after delete GET = %d; want %d", respGet.StatusCode, http.StatusNotFound)
	}
}

func TestUpdatePDFE2E(t *testing.T) {
	ts := setupServer(t)

	// ---- Step 1: Upload initial PDF ----
	initOut := initiateUpload(t, ts.URL, "sample.pdf")
	putPresignedFile(t, initOut.URL, "application/pdf", testutil.LoadPDF(t))
	finaliseUpload(t, ts.URL, initOut.ID, "docs")
	_ = waitOptimised(t, ts.URL, initOut.ID, false)

	// ---- Step 2: Initiate update ----
	updOut := initiateUpdate(t, ts.URL, initOut.ID, "sample-v2.pdf")

	// ---- Step 3: PUT updated PDF ----
	putPresignedFile(t, updOut.URL, "application/pdf", testutil.LoadPDF(t))

	// ---- Step 4: Finalise update ----
	finaliseUpdate(t, ts.URL, initOut.ID, "docs")

	// ---- Step 5: Poll GET media details until optimised ----
	getOut := waitOptimised(t, ts.URL, initOut.ID, false)

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
	} else if !strings.Contains(getOut.URL, fmt.Sprintf("%s-v2", initOut.ID)) {
		t.Errorf("URL = %q; want to contain version v2", getOut.URL)
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

	// ---- Step 6: Delete the media ----
	reqDel, err := http.NewRequest(http.MethodDelete, ts.URL+"/medias/"+initOut.ID, nil)
	if err != nil {
		t.Fatalf("new DELETE request error: %v", err)
	}
	respDel, err := http.DefaultClient.Do(reqDel)
	if err != nil {
		t.Fatalf("DELETE media error: %v", err)
	}
	respDel.Body.Close()
	if respDel.StatusCode != http.StatusNoContent {
		t.Fatalf("status delete = %d; want %d", respDel.StatusCode, http.StatusNoContent)
	}

	// ---- Step 7: Confirm GET returns 404 ----
	respGet, err := http.Get(ts.URL + "/medias/" + initOut.ID)
	if err != nil {
		t.Fatalf("GET after delete error: %v", err)
	}
	defer respGet.Body.Close()
	if respGet.StatusCode != http.StatusNotFound {
		t.Fatalf("status after delete GET = %d; want %d", respGet.StatusCode, http.StatusNotFound)
	}
}
