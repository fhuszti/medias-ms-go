package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"context"
	"github.com/fhuszti/medias-ms-go/internal/db"
	mediaUC "github.com/fhuszti/medias-ms-go/internal/usecase/media"
	"github.com/google/uuid"
)

type mockFinaliser struct {
	in  mediaUC.FinaliseUploadInput
	err error
}

func (m *mockFinaliser) FinaliseUpload(ctx context.Context, in mediaUC.FinaliseUploadInput) error {
	m.in = in
	return m.err
}

func TestFinaliseUploadHandler(t *testing.T) {
	validID := db.UUID(uuid.MustParse("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"))
	tests := []struct {
		name            string
		ctxID           bool
		body            string
		svcErr          error
		wantStatus      int
		wantContentType string
		wantErrorMap    map[string]string // for validation-error JSON
		wantBodyContain string            // substring for plain-text errors
	}{
		{
			name:            "missing ID",
			ctxID:           false,
			body:            `{"destBucket":"bucket1"}`,
			wantStatus:      http.StatusBadRequest,
			wantContentType: "application/json",
			wantBodyContain: "ID is required",
		},
		{
			name:            "invalid JSON",
			ctxID:           true,
			body:            `{"destBucket":`, // malformed
			wantStatus:      http.StatusBadRequest,
			wantContentType: "application/json",
			wantBodyContain: "invalid request payload",
		},
		{
			name:            "validation error: empty destBucket",
			ctxID:           true,
			body:            `{"destBucket":""}`,
			wantStatus:      http.StatusBadRequest,
			wantContentType: "application/json",
			wantErrorMap:    map[string]string{"destBucket": "required"},
		},
		{
			name:            "validation error: invalid destBucket",
			ctxID:           true,
			body:            `{"destBucket":"not-a-bucket"}`,
			wantStatus:      http.StatusBadRequest,
			wantContentType: "application/json",
			wantBodyContain: "destination bucket \"not-a-bucket\" does not exist",
		},
		{
			name:            "service error",
			ctxID:           true,
			body:            `{"destBucket":"bucket1"}`,
			svcErr:          errors.New("oops"),
			wantStatus:      http.StatusInternalServerError,
			wantContentType: "application/json",
			wantBodyContain: "could not finalise upload of media #" + validID.String(),
		},
		{
			name:            "happy path",
			ctxID:           true,
			body:            `{"destBucket":"bucket1"}`,
			wantStatus:      http.StatusNoContent,
			wantContentType: "",
		},
	}

	allowed := []string{"bucket1", "mydest"}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockSvc := &mockFinaliser{err: tc.svcErr}
			h := FinaliseUploadHandler(mockSvc, allowed)

			req := httptest.NewRequest(http.MethodPost, "/any", bytes.NewBufferString(tc.body))
			if tc.ctxID {
				req = req.WithContext(context.WithValue(
					req.Context(),
					IDKey,
					validID,
				))
			}
			rec := httptest.NewRecorder()

			h(rec, req)

			if rec.Code != tc.wantStatus {
				t.Fatalf("status = %d; want %d", rec.Code, tc.wantStatus)
			}
			if ct := rec.Header().Get("Content-Type"); ct != tc.wantContentType {
				t.Errorf("Content-Type = %q; want %q", ct, tc.wantContentType)
			}

			body := rec.Body.Bytes()
			if tc.wantStatus == http.StatusNoContent {
				if len(body) != 0 {
					t.Errorf("expected empty body, got %q", body)
				}
			} else if tc.wantErrorMap != nil {
				// unmarshal into map[string]string
				var errs map[string]string
				if err := json.Unmarshal(body, &errs); err != nil {
					t.Fatalf("invalid JSON error body: %v; body=%s", err, body)
				}
				for k, want := range tc.wantErrorMap {
					v, ok := errs[k]
					if !ok {
						t.Errorf("missing key %q in %v", k, errs)
					} else if !strings.Contains(v, want) {
						t.Errorf("errs[%q] = %q; want to contain %q", k, v, want)
					}
				}

			} else {
				// JSON error message
				var resp ErrorResponse
				if err := json.Unmarshal(body, &resp); err != nil {
					t.Fatalf("invalid JSON error body: %v; body=%s", err, body)
				}
				if !strings.Contains(resp.Error, tc.wantBodyContain) {
					t.Errorf("body = %q; want to contain %q", body, tc.wantBodyContain)
				}
			}

			// If the service was invoked, verify inputs
			// Only invoked when ctxID and JSON validation passed and no body error
			if tc.ctxID && tc.wantErrorMap == nil && tc.wantBodyContain == "" {
				if mockSvc.in.DestBucket != "bucket1" {
					t.Errorf("service got DestBucket = %q; want bucket1", mockSvc.in.DestBucket)
				}
				if mockSvc.in.ID != validID {
					t.Errorf("service got ID = %v; want %v", mockSvc.in.ID, validID)
				}
			}
		})
	}
}
