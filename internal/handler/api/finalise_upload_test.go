package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

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
		ctxBucket       string
		body            string
		svcErr          error
		wantStatus      int
		wantContentType string
		wantErrorMap    map[string]string // for validation-error JSON
		wantBodyContain string            // substring for plain-text errors
	}{
		{
			name:            "missing destBucket",
			ctxBucket:       "",
			body:            `{"id":"aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"}`,
			wantStatus:      http.StatusBadRequest,
			wantContentType: "application/json",
			wantBodyContain: "destination bucket is required",
		},
		{
			name:            "invalid JSON",
			ctxBucket:       "bucket1",
			body:            `{"id":`, // malformed
			wantStatus:      http.StatusBadRequest,
			wantContentType: "application/json",
			wantBodyContain: "invalid request payload",
		},
		{
			name:            "validation error: empty id",
			ctxBucket:       "bucket1",
			body:            `{"id":""}`,
			wantStatus:      http.StatusBadRequest,
			wantContentType: "application/json",
			wantErrorMap:    map[string]string{"id": "required"},
		},
		{
			name:            "validation error: bad id",
			ctxBucket:       "bucket1",
			body:            `{"id":"not-a-uuid"}`,
			wantStatus:      http.StatusBadRequest,
			wantContentType: "application/json",
			wantErrorMap:    map[string]string{"id": "uuid"},
		},
		{
			name:            "service error",
			ctxBucket:       "bucket1",
			body:            `{"id":"aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"}`,
			svcErr:          errors.New("oops"),
			wantStatus:      http.StatusInternalServerError,
			wantContentType: "application/json",
			wantBodyContain: "could not finalise upload of media #" + validID.String(),
		},
		{
			name:            "happy path",
			ctxBucket:       "mydest",
			body:            `{"id":"aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"}`,
			wantStatus:      http.StatusNoContent,
			wantContentType: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockSvc := &mockFinaliser{err: tc.svcErr}
			h := FinaliseUploadHandler(mockSvc)

			req := httptest.NewRequest(http.MethodPost, "/any", bytes.NewBufferString(tc.body))
			if tc.ctxBucket != "" {
				req = req.WithContext(context.WithValue(
					req.Context(),
					DestBucketKey,
					tc.ctxBucket,
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
				// plain-text error
				if !bytes.Contains(body, []byte(tc.wantBodyContain)) {
					t.Errorf("body = %q; want to contain %q", body, tc.wantBodyContain)
				}
			}

			// If the service was invoked, verify inputs
			// Only invoked when ctxBucket non-empty AND JSON validation passed
			if tc.ctxBucket != "" && tc.wantErrorMap == nil && tc.wantBodyContain == "" {
				if mockSvc.in.DestBucket != tc.ctxBucket {
					t.Errorf("service got DestBucket = %q; want %q", mockSvc.in.DestBucket, tc.ctxBucket)
				}
				if mockSvc.in.ID != validID {
					t.Errorf("service got ID = %v; want %v", mockSvc.in.ID, validID)
				}
			}
		})
	}
}
