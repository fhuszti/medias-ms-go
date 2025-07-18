package api

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/fhuszti/medias-ms-go/internal/mock"
	mediaUC "github.com/fhuszti/medias-ms-go/internal/usecase/media"
	msuuid "github.com/fhuszti/medias-ms-go/internal/uuid"
	guuid "github.com/google/uuid"
)

func TestDeleteMediaHandler(t *testing.T) {
	validID := msuuid.UUID(guuid.MustParse("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"))
	tests := []struct {
		name           string
		ctxID          *msuuid.UUID
		svcErr         error
		wantStatus     int
		wantBodySubstr string
	}{
		{
			name:           "missing id",
			ctxID:          nil,
			svcErr:         nil,
			wantStatus:     http.StatusBadRequest,
			wantBodySubstr: "ID is required",
		},
		{
			name:           "not found",
			ctxID:          &validID,
			svcErr:         mediaUC.ErrObjectNotFound,
			wantStatus:     http.StatusNotFound,
			wantBodySubstr: "Media not found",
		},
		{
			name:           "service error",
			ctxID:          &validID,
			svcErr:         errors.New("boom"),
			wantStatus:     http.StatusInternalServerError,
			wantBodySubstr: "Failed to delete media",
		},
		{
			name:       "happy path",
			ctxID:      &validID,
			svcErr:     nil,
			wantStatus: http.StatusNoContent,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockSvc := &mock.MediaDeleter{Err: tc.svcErr}
			h := DeleteMediaHandler(mockSvc)

			req := httptest.NewRequest(http.MethodDelete, "/medias/"+validID.String(), nil)
			if tc.ctxID != nil {
				req = req.WithContext(context.WithValue(req.Context(), IDKey, *tc.ctxID))
			}

			rec := httptest.NewRecorder()
			h(rec, req)

			if rec.Code != tc.wantStatus {
				t.Fatalf("status = %d; want %d", rec.Code, tc.wantStatus)
			}

			if tc.wantStatus == http.StatusNoContent {
				if rec.Body.Len() != 0 {
					t.Errorf("expected empty body, got %q", rec.Body.String())
				}
				if mockSvc.ID != validID {
					t.Errorf("service got ID = %s; want %s", mockSvc.ID, validID)
				}
			} else {
				if !errors.Is(tc.svcErr, mediaUC.ErrObjectNotFound) && tc.ctxID != nil {
					if mockSvc.ID != validID {
						t.Errorf("service got ID = %s; want %s", mockSvc.ID, validID)
					}
				}
				if !contains(rec.Body.String(), tc.wantBodySubstr) {
					t.Errorf("body = %q; want to contain %q", rec.Body.String(), tc.wantBodySubstr)
				}
			}
		})
	}
}

func contains(haystack, needle string) bool {
	return needle == "" || strings.Contains(haystack, needle)
}
