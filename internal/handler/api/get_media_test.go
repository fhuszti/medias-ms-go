package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"github.com/fhuszti/medias-ms-go/internal/db"
	"github.com/google/uuid"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/fhuszti/medias-ms-go/internal/model"
	mediaSvc "github.com/fhuszti/medias-ms-go/internal/usecase/media"
)

type mockGetter struct {
	out mediaSvc.GetMediaOutput
	err error
	in  mediaSvc.GetMediaInput
}

func (m *mockGetter) GetMedia(ctx context.Context, in mediaSvc.GetMediaInput) (*mediaSvc.GetMediaOutput, error) {
	m.in = in
	return &m.out, m.err
}

func TestGetMediaHandler(t *testing.T) {
	validID := db.UUID(uuid.MustParse("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"))
	nonEmptyVariants := model.VariantsOutput{
		model.VariantOutput{Width: 100, Height: 50, SizeBytes: 1234, URL: "https://cdn.example.com/foo_100"},
	}

	tests := []struct {
		name             string
		ctxID            *db.UUID
		svcOut           mediaSvc.GetMediaOutput
		svcErr           error
		wantStatus       int
		wantContentType  string
		wantCacheControl string

		wantOutput       *mediaSvc.GetMediaOutput
		wantBodyContains string
	}{
		{
			name:  "happy path uses cache",
			ctxID: &validID,
			svcOut: mediaSvc.GetMediaOutput{
				ValidUntil: time.Now(),
				Optimised:  true,
				URL:        "https://cdn.example.com/foo",
				Metadata:   mediaSvc.MetadataOutput{},
				Variants:   nonEmptyVariants,
			},
			svcErr:           nil,
			wantStatus:       http.StatusOK,
			wantContentType:  "application/json",
			wantCacheControl: "public, max-age=1200",
			wantOutput:       &mediaSvc.GetMediaOutput{},
		},
		{
			name:  "optimised true for image but no variants → no cache",
			ctxID: &validID,
			svcOut: mediaSvc.GetMediaOutput{
				ValidUntil: time.Now(),
				Optimised:  true,
				URL:        "https://cdn.example.com/presigned",
				Metadata:   mediaSvc.MetadataOutput{MimeType: "image/png"},
				Variants:   model.VariantsOutput{}, // no variants
			},
			svcErr:           nil,
			wantStatus:       http.StatusOK,
			wantContentType:  "application/json",
			wantCacheControl: "no-store, max-age=0, must-revalidate",
			wantOutput:       &mediaSvc.GetMediaOutput{},
		},
		{
			name:  "optimised false for image but has variants → no cache",
			ctxID: &validID,
			svcOut: mediaSvc.GetMediaOutput{
				ValidUntil: time.Now(),
				Optimised:  false,
				URL:        "https://cdn.example.com/presigned",
				Metadata:   mediaSvc.MetadataOutput{MimeType: "image/png"},
				Variants:   nonEmptyVariants, // variants present
			},
			svcErr:           nil,
			wantStatus:       http.StatusOK,
			wantContentType:  "application/json",
			wantCacheControl: "no-store, max-age=0, must-revalidate",
			wantOutput:       &mediaSvc.GetMediaOutput{},
		},
		{
			name:             "service error",
			ctxID:            &validID,
			svcOut:           mediaSvc.GetMediaOutput{},
			svcErr:           errors.New("boom"),
			wantStatus:       http.StatusInternalServerError,
			wantContentType:  "application/json",
			wantCacheControl: "no-store, max-age=0, must-revalidate",
			wantBodyContains: "Could not get media details",
		},
		{
			name:             "missing ID",
			ctxID:            nil,
			svcOut:           mediaSvc.GetMediaOutput{},
			svcErr:           nil,
			wantStatus:       http.StatusBadRequest,
			wantContentType:  "application/json",
			wantBodyContains: "ID is required",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockSvc := &mockGetter{
				out: tc.svcOut,
				err: tc.svcErr,
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

			switch {
			case tc.wantOutput != nil:
				// decode into your output struct
				dec := json.NewDecoder(bytes.NewReader(rec.Body.Bytes()))
				dec.DisallowUnknownFields()
				if err := dec.Decode(tc.wantOutput); err != nil {
					t.Fatalf("JSON decode = %v (body=%q)", err, rec.Body.String())
				}
				// verify service was called with the correct ID
				if mockSvc.in.ID != *tc.ctxID {
					t.Errorf("service got ID = %s; want %s", mockSvc.in.ID, *tc.ctxID)
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
