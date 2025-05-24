package media

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
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

func (m *mockGetter) GetMedia(ctx context.Context, in mediaSvc.GetMediaInput) (mediaSvc.GetMediaOutput, error) {
	m.in = in
	return m.out, m.err
}

func TestGetMediaHandler(t *testing.T) {
	happyOutput := mediaSvc.GetMediaOutput{
		ValidUntil: time.Now(),
		Optimised:  true,
		URL:        "https://cdn.example.com/presigned",
		Metadata:   mediaSvc.MetadataOutput{},
		Variants:   model.VariantsOutput{},
	}

	tests := []struct {
		name            string
		body            string
		svcOut          mediaSvc.GetMediaOutput
		svcErr          error
		wantStatus      int
		wantContentType string

		wantOutput       *mediaSvc.GetMediaOutput
		wantErrorMap     map[string]string
		wantBodyContains string
	}{
		{
			name:            "happy path",
			body:            `{"id":"aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"}`,
			svcOut:          happyOutput,
			svcErr:          nil,
			wantStatus:      http.StatusCreated,
			wantContentType: "application/json",
			wantOutput:      &mediaSvc.GetMediaOutput{},
		},
		{
			name:             "invalid JSON",
			body:             `{"id":`, // malformed
			svcOut:           mediaSvc.GetMediaOutput{},
			svcErr:           nil,
			wantStatus:       http.StatusBadRequest,
			wantContentType:  "application/json",
			wantBodyContains: "Invalid request",
		},
		{
			name:            "validation error: empty id",
			body:            `{"id":""}`,
			svcOut:          mediaSvc.GetMediaOutput{},
			svcErr:          nil,
			wantStatus:      http.StatusBadRequest,
			wantContentType: "application/json",
			wantErrorMap:    map[string]string{"id": "required"},
		},
		{
			name:            "validation error: bad id",
			body:            `{"id":"not-uuid"}`,
			svcOut:          mediaSvc.GetMediaOutput{},
			svcErr:          nil,
			wantStatus:      http.StatusBadRequest,
			wantContentType: "application/json",
			wantErrorMap:    map[string]string{"id": "uuid"},
		},
		{
			name:             "service error",
			body:             `{"id":"aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"}`,
			svcOut:           mediaSvc.GetMediaOutput{},
			svcErr:           errors.New("boom"),
			wantStatus:       http.StatusInternalServerError,
			wantContentType:  "application/json",
			wantBodyContains: "Could not get media details",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockSvc := &mockGetter{
				out: tc.svcOut,
				err: tc.svcErr,
			}
			handlerFn := GetMediaHandler(mockSvc)

			req := httptest.NewRequest(http.MethodPost, "/get_media", strings.NewReader(tc.body))
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
