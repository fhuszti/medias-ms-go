package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/fhuszti/medias-ms-go/internal/mock"
	"github.com/fhuszti/medias-ms-go/internal/port"
	msuuid "github.com/fhuszti/medias-ms-go/internal/uuid"
	guuid "github.com/google/uuid"
)

func TestGenerateUploadLinkHandler(t *testing.T) {
	tests := []struct {
		name            string
		body            string
		svcOut          port.GenerateUploadLinkOutput
		svcErr          error
		wantStatus      int
		wantContentType string

		wantOutput       *port.GenerateUploadLinkOutput
		wantErrorMap     map[string]string
		wantBodyContains string
	}{
		{
			name:            "happy path",
			body:            `{"name":"my-file.png"}`,
			svcOut:          port.GenerateUploadLinkOutput{ID: msuuid.UUID(guuid.MustParse("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee")), URL: "https://cdn.example.com/presigned"},
			svcErr:          nil,
			wantStatus:      http.StatusCreated,
			wantContentType: "application/json",
			wantOutput:      &port.GenerateUploadLinkOutput{},
		},
		{
			name:             "invalid JSON",
			body:             `{"name":`, // malformed
			svcOut:           port.GenerateUploadLinkOutput{},
			svcErr:           nil,
			wantStatus:       http.StatusBadRequest,
			wantContentType:  "application/json",
			wantBodyContains: "Invalid request",
		},
		{
			name:            "validation error: empty name",
			body:            `{"name":""}`,
			svcOut:          port.GenerateUploadLinkOutput{},
			svcErr:          nil,
			wantStatus:      http.StatusBadRequest,
			wantContentType: "application/json",
			wantErrorMap:    map[string]string{"name": "required"},
		},
		{
			name:            "validation error: name too long",
			body:            fmt.Sprintf(`{"name":"%s"}`, strings.Repeat("a", 81)),
			svcOut:          port.GenerateUploadLinkOutput{},
			svcErr:          nil,
			wantStatus:      http.StatusBadRequest,
			wantContentType: "application/json",
			wantErrorMap:    map[string]string{"name": "max"},
		},
		{
			name:             "service error",
			body:             `{"name":"ok.png"}`,
			svcOut:           port.GenerateUploadLinkOutput{},
			svcErr:           errors.New("boom"),
			wantStatus:       http.StatusInternalServerError,
			wantContentType:  "application/json",
			wantBodyContains: "Could not generate upload link",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockSvc := &mock.UploadLinkGenerator{
				Out: tc.svcOut,
				Err: tc.svcErr,
			}
			handlerFn := GenerateUploadLinkHandler(mockSvc)

			req := httptest.NewRequest(http.MethodPost, "/upload", strings.NewReader(tc.body))
			// we don't need any special headers; JSON decoder uses Body only

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
				// decode into your output struct
				dec := json.NewDecoder(bytes.NewReader(data))
				dec.DisallowUnknownFields()
				if err := dec.Decode(tc.wantOutput); err != nil {
					t.Fatalf("JSON decode = %v (body=%q)", err, string(data))
				}
				// then assert each field:
				if got, want := tc.wantOutput.ID, tc.svcOut.ID; got != want {
					t.Errorf("ID = %v; want %v", got, want)
				}
				if got, want := tc.wantOutput.URL, tc.svcOut.URL; got != want {
					t.Errorf("URL = %q; want %q", got, want)
				}

			case tc.wantErrorMap != nil:
				// unmarshal into a map and compare
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
		})
	}
}
