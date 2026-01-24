package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/fhuszti/medias-ms-go/internal/api_context"
	"github.com/fhuszti/medias-ms-go/internal/mock"
	"github.com/fhuszti/medias-ms-go/internal/port"
	"github.com/fhuszti/medias-ms-go/internal/usecase/media"
	msuuid "github.com/fhuszti/medias-ms-go/internal/uuid"
	guuid "github.com/google/uuid"
)

func TestInitiateUpdateHandler(t *testing.T) {
	validID := msuuid.UUID(guuid.MustParse("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"))
	tests := []struct {
		name            string
		ctxID           bool
		body            string
		svcOut          port.InitiateUpdateOutput
		svcErr          error
		wantStatus      int
		wantContentType string
		wantInputName   string

		wantOutput       *port.InitiateUpdateOutput
		wantErrorMap     map[string]string
		wantBodyContains string
	}{
		{
			name:             "missing ID",
			ctxID:            false,
			body:             `{"name":"my-file.png"}`,
			svcOut:           port.InitiateUpdateOutput{},
			wantStatus:       http.StatusBadRequest,
			wantContentType:  "application/json",
			wantBodyContains: "ID is required",
		},
		{
			name:             "invalid JSON",
			ctxID:            true,
			body:             `{"name":`, // malformed
			svcOut:           port.InitiateUpdateOutput{},
			wantStatus:       http.StatusBadRequest,
			wantContentType:  "application/json",
			wantBodyContains: "Invalid request",
		},
		{
			name:            "validation error: empty name",
			ctxID:           true,
			body:            `{"name":""}`,
			svcOut:          port.InitiateUpdateOutput{},
			wantStatus:      http.StatusBadRequest,
			wantContentType: "application/json",
			wantErrorMap:    map[string]string{"name": "required"},
		},
		{
			name:            "validation error: name too long",
			ctxID:           true,
			body:            fmt.Sprintf(`{"name":"%s"}`, strings.Repeat("a", 81)),
			svcOut:          port.InitiateUpdateOutput{},
			wantStatus:      http.StatusBadRequest,
			wantContentType: "application/json",
			wantErrorMap:    map[string]string{"name": "max"},
		},
		{
			name:             "media not found",
			ctxID:            true,
			body:             `{"name":"ok.png"}`,
			svcOut:           port.InitiateUpdateOutput{},
			svcErr:           media.ErrObjectNotFound,
			wantStatus:       http.StatusNotFound,
			wantContentType:  "application/json",
			wantBodyContains: "Media not found",
		},
		{
			name:             "service error",
			ctxID:            true,
			body:             `{"name":"ok.png"}`,
			svcOut:           port.InitiateUpdateOutput{},
			svcErr:           errors.New("boom"),
			wantStatus:       http.StatusInternalServerError,
			wantContentType:  "application/json",
			wantBodyContains: "Could not initiate update",
		},
		{
			name:            "happy path",
			ctxID:           true,
			body:            `{"name":"my-file.png"}`,
			svcOut:          port.InitiateUpdateOutput{ID: validID, URL: "https://cdn.example.com/presigned"},
			wantStatus:      http.StatusCreated,
			wantContentType: "application/json",
			wantInputName:   "my-file.png",
			wantOutput:      &port.InitiateUpdateOutput{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockSvc := &mock.UpdateInitiator{
				Out: tc.svcOut,
				Err: tc.svcErr,
			}
			handlerFn := InitiateUpdateHandler(mockSvc)

			req := httptest.NewRequest(http.MethodPost, "/update", strings.NewReader(tc.body))
			if tc.ctxID {
				req = req.WithContext(context.WithValue(
					req.Context(),
					api_context.IDKey,
					validID,
				))
			}
			rec := httptest.NewRecorder()

			handlerFn(rec, req)

			if rec.Code != tc.wantStatus {
				t.Fatalf("status = %d; want %d", rec.Code, tc.wantStatus)
			}

			gotCT := rec.Header().Get("Content-Type")
			if gotCT != tc.wantContentType {
				t.Errorf("Content-Type = %q; want %q", gotCT, tc.wantContentType)
			}

			data := rec.Body.Bytes()

			switch {
			case tc.wantOutput != nil:
				dec := json.NewDecoder(bytes.NewReader(data))
				dec.DisallowUnknownFields()
				if err := dec.Decode(tc.wantOutput); err != nil {
					t.Fatalf("JSON decode = %v (body=%q)", err, string(data))
				}
				if got, want := tc.wantOutput.ID, tc.svcOut.ID; got != want {
					t.Errorf("ID = %v; want %v", got, want)
				}
				if got, want := tc.wantOutput.URL, tc.svcOut.URL; got != want {
					t.Errorf("URL = %q; want %q", got, want)
				}

			case tc.wantErrorMap != nil:
				var errs map[string]string
				if err := json.Unmarshal(data, &errs); err != nil {
					t.Fatalf("error JSON: %v; body=%q", err, string(data))
				}
				for k, want := range tc.wantErrorMap {
					if got, ok := errs[k]; !ok {
						t.Errorf("missing key %q in error response: %v", k, errs)
					} else if got != want {
						t.Errorf("errs[%q] = %q; want %q", k, got, want)
					}
				}

			case tc.wantBodyContains != "":
				if !strings.Contains(string(data), tc.wantBodyContains) {
					t.Errorf("body = %q; want to contain %q", string(data), tc.wantBodyContains)
				}

			default:
				t.Fatal("test case has no assertion target!")
			}

			if tc.ctxID && tc.wantOutput != nil {
				if mockSvc.In.ID != validID {
					t.Errorf("service got ID = %v; want %v", mockSvc.In.ID, validID)
				}
				if tc.wantInputName != "" && mockSvc.In.Name != tc.wantInputName {
					t.Errorf("service got Name = %q; want %q", mockSvc.In.Name, tc.wantInputName)
				}
			}
		})
	}
}
