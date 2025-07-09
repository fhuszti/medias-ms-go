package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"hash/crc32"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/fhuszti/medias-ms-go/internal/mock"
	"github.com/fhuszti/medias-ms-go/internal/model"
	"github.com/fhuszti/medias-ms-go/internal/port"
	msuuid "github.com/fhuszti/medias-ms-go/internal/uuid"
	guuid "github.com/google/uuid"
)

func computeETag(t testing.TB, v any) string {
	t.Helper()
	raw, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return fmt.Sprintf("\"%08x\"", crc32.ChecksumIEEE(raw))
}

func TestGetMediaHandler(t *testing.T) {
	validID := msuuid.UUID(guuid.MustParse("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"))
	nonEmptyVariants := model.VariantsOutput{
		model.VariantOutput{Width: 100, Height: 50, SizeBytes: 1234, URL: "https://cdn.example.com/foo_100"},
	}

	tests := []struct {
		name             string
		ctxID            *msuuid.UUID
		svcOut           port.GetMediaOutput
		svcErr           error
		wantStatus       int
		wantContentType  string
		wantCacheControl string
		wantETag         bool

		wantOutput       *port.GetMediaOutput
		wantBodyContains string
	}{
		{
			name:  "happy path uses cache",
			ctxID: &validID,
			svcOut: port.GetMediaOutput{
				ValidUntil: time.Now(),
				Optimised:  true,
				URL:        "https://cdn.example.com/foo",
				Metadata:   port.MetadataOutput{},
				Variants:   nonEmptyVariants,
			},
			svcErr:           nil,
			wantStatus:       http.StatusOK,
			wantContentType:  "application/json",
			wantCacheControl: "max-age=0",
			wantETag:         true,
			wantOutput:       &port.GetMediaOutput{},
		},
		{
			name:  "optimised true for image but no variants → no cache",
			ctxID: &validID,
			svcOut: port.GetMediaOutput{
				ValidUntil: time.Now(),
				Optimised:  true,
				URL:        "https://cdn.example.com/presigned",
				Metadata:   port.MetadataOutput{MimeType: "image/png"},
				Variants:   model.VariantsOutput{}, // no variants
			},
			svcErr:           nil,
			wantStatus:       http.StatusOK,
			wantContentType:  "application/json",
			wantCacheControl: "max-age=0",
			wantETag:         true,
			wantOutput:       &port.GetMediaOutput{},
		},
		{
			name:  "optimised false for image but has variants → no cache",
			ctxID: &validID,
			svcOut: port.GetMediaOutput{
				ValidUntil: time.Now(),
				Optimised:  false,
				URL:        "https://cdn.example.com/presigned",
				Metadata:   port.MetadataOutput{MimeType: "image/png"},
				Variants:   nonEmptyVariants, // variants present
			},
			svcErr:           nil,
			wantStatus:       http.StatusOK,
			wantContentType:  "application/json",
			wantCacheControl: "max-age=0",
			wantETag:         true,
			wantOutput:       &port.GetMediaOutput{},
		},
		{
			name:             "service error",
			ctxID:            &validID,
			svcOut:           port.GetMediaOutput{},
			svcErr:           errors.New("boom"),
			wantStatus:       http.StatusInternalServerError,
			wantContentType:  "application/json",
			wantCacheControl: "no-store, max-age=0, must-revalidate",
			wantBodyContains: "Could not get media details",
		},
		{
			name:             "missing ID",
			ctxID:            nil,
			svcOut:           port.GetMediaOutput{},
			svcErr:           nil,
			wantStatus:       http.StatusBadRequest,
			wantContentType:  "application/json",
			wantBodyContains: "ID is required",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockSvc := &mock.MediaGetter{
				Out: &tc.svcOut,
				Err: tc.svcErr,
			}
			handlerFn := GetMediaHandler(mockSvc)

			req := httptest.NewRequest(http.MethodPost, "/medias/"+validID.String(), nil)
			if tc.ctxID != nil {
				req = req.WithContext(context.WithValue(req.Context(), IDKey, *tc.ctxID))
			}
			rec := httptest.NewRecorder()

			handlerFn(rec, req)

			if rec.Code != tc.wantStatus {
				t.Fatalf("status = %d; want %d", rec.Code, tc.wantStatus)
			}
			if ct := rec.Header().Get("Content-Type"); ct != tc.wantContentType {
				t.Errorf("Content-Type = %q; want %q", ct, tc.wantContentType)
			}
			if tc.wantCacheControl != "" {
				if cc := rec.Header().Get("Cache-Control"); cc != tc.wantCacheControl {
					t.Errorf("Cache-Control = %q; want %q", cc, tc.wantCacheControl)
				}
			}
			if tc.wantETag {
				wantETag := computeETag(t, tc.svcOut)
				if et := rec.Header().Get("ETag"); et != wantETag {
					t.Errorf("ETag = %q; want %q", et, wantETag)
				}
			}

			switch {
			case tc.wantOutput != nil:
				// decode into your output struct
				dec := json.NewDecoder(bytes.NewReader(rec.Body.Bytes()))
				dec.DisallowUnknownFields()
				if err := dec.Decode(tc.wantOutput); err != nil {
					t.Fatalf("JSON decode = %v (body=%q)", err, rec.Body.String())
				}
				// verify service was called with the correct ID
				if mockSvc.Id != *tc.ctxID {
					t.Errorf("service got ID = %s; want %s", mockSvc.Id, *tc.ctxID)
				}
				// verify URL field
				if got, want := tc.wantOutput.URL, tc.svcOut.URL; got != want {
					t.Errorf("URL = %q; want %q", got, want)
				}
			case tc.wantBodyContains != "":
				if !strings.Contains(rec.Body.String(), tc.wantBodyContains) {
					t.Errorf("body = %q; want to contain %q", rec.Body.String(), tc.wantBodyContains)
				}
			default:
				t.Fatal("test case has no assertion target!")
			}
		})
	}
}

func TestGetMediaHandler_IfNoneMatch(t *testing.T) {
	validID := msuuid.UUID(guuid.MustParse("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"))
	mockSvc := &mock.MediaGetter{
		Out: &port.GetMediaOutput{
			ValidUntil: time.Now(),
			Optimised:  true,
			URL:        "https://cdn.example.com/foo",
			Metadata:   port.MetadataOutput{},
		},
		Err: nil,
	}

	handlerFn := GetMediaHandler(mockSvc)
	etag := computeETag(t, mockSvc.Out)
	req := httptest.NewRequest(http.MethodGet, "/medias/"+validID.String(), nil)
	req = req.WithContext(context.WithValue(req.Context(), IDKey, validID))
	req.Header.Set("If-None-Match", etag)
	rec := httptest.NewRecorder()

	handlerFn(rec, req)

	if rec.Code != http.StatusNotModified {
		t.Fatalf("status = %d; want %d", rec.Code, http.StatusNotModified)
	}
	if et := rec.Header().Get("ETag"); et != etag {
		t.Errorf("ETag = %q; want %q", et, etag)
	}
	if cc := rec.Header().Get("Cache-Control"); cc != "max-age=0" {
		t.Errorf("Cache-Control = %q; want %q", cc, "max-age=0")
	}
	if rec.Body.Len() != 0 {
		t.Errorf("expected empty body, got %q", rec.Body.String())
	}
}
