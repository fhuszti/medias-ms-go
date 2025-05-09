package media

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	mediaService "github.com/fhuszti/medias-ms-go/internal/usecase/media"
)

type fakeService struct {
	outURL string
	outErr error
	called bool
	in     mediaService.GenerateUploadLinkInput
}

func (f *fakeService) GenerateUploadLink(ctx context.Context, in mediaService.GenerateUploadLinkInput) (string, error) {
	f.called = true
	f.in = in
	return f.outURL, f.outErr
}

func TestGenerateUploadLinkHandler_InvalidJSON(t *testing.T) {
	fsvc := &fakeService{}
	h := GenerateUploadLinkHandler(fsvc)

	req := httptest.NewRequest("POST", "/", strings.NewReader("{not json"))
	rec := httptest.NewRecorder()
	h(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d; want %d", rec.Code, http.StatusBadRequest)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "Invalid request:") {
		t.Errorf("body = %q; want to contain %q", body, "Invalid request:")
	}
	if fsvc.called {
		t.Error("service should not have been called on invalid JSON")
	}
}

func TestGenerateUploadLinkHandler_ValidationError(t *testing.T) {
	fsvc := &fakeService{}
	h := GenerateUploadLinkHandler(fsvc)

	// Missing "name", invalid "type"
	payload := `{"type":"application/x-foo"}`
	req := httptest.NewRequest("POST", "/", strings.NewReader(payload))
	rec := httptest.NewRecorder()
	h(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d; want %d", rec.Code, http.StatusBadRequest)
	}
	body := strings.TrimSpace(rec.Body.String())
	// Should be a JSON object mapping "name" → "required" or "type"→"mimetype"
	var errs map[string]string
	if err := json.Unmarshal([]byte(body), &errs); err != nil {
		t.Fatalf("response body is not valid JSON map: %v", err)
	}
	if errs["name"] != "required" {
		t.Errorf(`errs["name"] = %q; want "required"`, errs["name"])
	}
	if fsvc.called {
		t.Error("service should not have been called on validation error")
	}
}

func TestGenerateUploadLinkHandler_ServiceError(t *testing.T) {
	fsvc := &fakeService{outErr: errors.New("boom")}
	h := GenerateUploadLinkHandler(fsvc)

	// Valid payload
	payload := `{"name":"foo","type":"image/png"}`
	req := httptest.NewRequest("POST", "/", strings.NewReader(payload))
	rec := httptest.NewRecorder()
	h(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d; want %d", rec.Code, http.StatusInternalServerError)
	}
	if !strings.Contains(rec.Body.String(), "Could not generate presigned URL for upload: boom") {
		t.Errorf("body = %q; want to contain service error", rec.Body.String())
	}
	if !fsvc.called {
		t.Error("service should have been called once")
	}
}

func TestGenerateUploadLinkHandler_Success(t *testing.T) {
	const wantURL = "https://cdn/upload/123"
	fsvc := &fakeService{outURL: wantURL}
	h := GenerateUploadLinkHandler(fsvc)

	payload := `{"name":"myfile","type":"image/png"}`
	req := httptest.NewRequest("POST", "/", strings.NewReader(payload))
	rec := httptest.NewRecorder()
	h(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("status = %d; want %d", rec.Code, http.StatusCreated)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q; want application/json", ct)
	}

	var got string
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decoding response JSON: %v", err)
	}
	if got != wantURL {
		t.Errorf("body = %q; want %q", got, wantURL)
	}

	// Make sure the service got the right input
	if !fsvc.called {
		t.Fatal("service was not called")
	}
	if fsvc.in.Name != "myfile" {
		t.Errorf("service in.Name = %q; want %q", fsvc.in.Name, "myfile")
	}
	if fsvc.in.Type != "image/png" {
		t.Errorf("service in.Type = %q; want %q", fsvc.in.Type, "image/png")
	}
}
