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

	"github.com/fhuszti/medias-ms-go/internal/api_context"
	"github.com/fhuszti/medias-ms-go/internal/mock"
	"github.com/fhuszti/medias-ms-go/internal/usecase/media"
	msuuid "github.com/fhuszti/medias-ms-go/internal/uuid"
	guuid "github.com/google/uuid"
)

func TestFinaliseUpdateHandler(t *testing.T) {
	validID := msuuid.UUID(guuid.MustParse("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"))
	tests := []struct {
		name            string
		ctxID           bool
		body            string
		svcErr          error
		wantStatus      int
		wantContentType string
		wantErrorMap    map[string]string
		wantBodyContain string
	}{
		{
			name:            "missing ID",
			ctxID:           false,
			body:            `{"dest_bucket":"bucket1"}`,
			wantStatus:      http.StatusBadRequest,
			wantContentType: "application/json",
			wantBodyContain: "ID is required",
		},
		{
			name:            "invalid JSON",
			ctxID:           true,
			body:            `{"dest_bucket":`,
			wantStatus:      http.StatusBadRequest,
			wantContentType: "application/json",
			wantBodyContain: "invalid request payload",
		},
		{
			name:            "validation error: empty dest_bucket",
			ctxID:           true,
			body:            `{"dest_bucket":""}`,
			wantStatus:      http.StatusBadRequest,
			wantContentType: "application/json",
			wantErrorMap:    map[string]string{"dest_bucket": "required"},
		},
		{
			name:            "validation error: invalid dest_bucket",
			ctxID:           true,
			body:            `{"dest_bucket":"not-a-bucket"}`,
			wantStatus:      http.StatusBadRequest,
			wantContentType: "application/json",
			wantBodyContain: "destination bucket \"not-a-bucket\" does not exist",
		},
		{
			name:            "media not found",
			ctxID:           true,
			body:            `{"dest_bucket":"bucket1"}`,
			svcErr:          media.ErrObjectNotFound,
			wantStatus:      http.StatusNotFound,
			wantContentType: "application/json",
			wantBodyContain: "Media not found",
		},
		{
			name:            "service error",
			ctxID:           true,
			body:            `{"dest_bucket":"bucket1"}`,
			svcErr:          errors.New("oops"),
			wantStatus:      http.StatusInternalServerError,
			wantContentType: "application/json",
			wantBodyContain: "could not finalise update of media #" + validID.String(),
		},
		{
			name:            "happy path",
			ctxID:           true,
			body:            `{"dest_bucket":"bucket1"}`,
			wantStatus:      http.StatusNoContent,
			wantContentType: "",
		},
	}

	allowed := []string{"bucket1", "mydest"}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockSvc := &mock.UpdateFinaliser{Err: tc.svcErr}
			h := FinaliseUpdateHandler(mockSvc, allowed)

			req := httptest.NewRequest(http.MethodPost, "/any", bytes.NewBufferString(tc.body))
			if tc.ctxID {
				req = req.WithContext(context.WithValue(
					req.Context(),
					api_context.IDKey,
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
				var resp ErrorResponse
				if err := json.Unmarshal(body, &resp); err != nil {
					t.Fatalf("invalid JSON error body: %v; body=%s", err, body)
				}
				if !strings.Contains(resp.Error, tc.wantBodyContain) {
					t.Errorf("body = %q; want to contain %q", body, tc.wantBodyContain)
				}
			}

			if tc.ctxID && tc.wantErrorMap == nil && tc.wantBodyContain == "" {
				if mockSvc.In.DestBucket != "bucket1" {
					t.Errorf("service got DestBucket = %q; want bucket1", mockSvc.In.DestBucket)
				}
				if mockSvc.In.ID != validID {
					t.Errorf("service got ID = %v; want %v", mockSvc.In.ID, validID)
				}
			}
		})
	}
}
