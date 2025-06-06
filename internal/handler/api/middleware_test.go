package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

func TestWithDestBucketMiddleware(t *testing.T) {
	allowed := []string{"staging", "images"}
	mw := WithDestBucket(allowed)

	tests := []struct {
		name           string
		paramValue     string // what chi.URLParam(r, "destBucket") returns
		wantStatus     int
		expectNextCall bool // if the next handler should run
	}{
		{"missing param", "", http.StatusBadRequest, false},
		{"forbidden bucket", "other", http.StatusBadRequest, false},
		{"allowed staging", "staging", http.StatusNoContent, true},
		{"allowed images", "images", http.StatusNoContent, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// dummy handler that records if it's called
			nextCalled := false
			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				nextCalled = true
				// echo back the bucket from context
				if b, ok := BucketFromContext(r.Context()); ok {
					w.Header().Set("X-Bucket", b)
				}
				w.WriteHeader(http.StatusNoContent)
			})

			req := httptest.NewRequest("GET", "/any", nil)
			// inject chi URLParam
			rctx := chi.NewRouteContext()
			if tc.paramValue != "" {
				rctx.URLParams.Add("destBucket", tc.paramValue)
			}
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			rec := httptest.NewRecorder()

			// call middleware
			handler := mw(next)
			handler.ServeHTTP(rec, req)

			if rec.Code != tc.wantStatus {
				t.Errorf("status = %d; want %d", rec.Code, tc.wantStatus)
			}
			if nextCalled != tc.expectNextCall {
				t.Errorf("nextCalled = %v; want %v", nextCalled, tc.expectNextCall)
			}
			if tc.expectNextCall {
				got := rec.Header().Get("X-Bucket")
				if got != tc.paramValue {
					t.Errorf("bucket in context = %q; want %q", got, tc.paramValue)
				}
			}
		})
	}
}

func TestWithIDMiddleware(t *testing.T) {
	mw := WithID()

	tests := []struct {
		name           string
		paramValue     string // what chi.URLParam(r, "id") returns
		wantStatus     int
		expectNextCall bool // if the next handler should run
	}{
		{"missing param", "", http.StatusBadRequest, false},
		{"bad param", "not-uuid", http.StatusBadRequest, false},
		{"happy path", "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee", http.StatusNoContent, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// dummy handler that records if it's called
			nextCalled := false
			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				nextCalled = true
				// echo back the bucket from context
				if id, ok := IDFromContext(r.Context()); ok {
					w.Header().Set("X-ID", id.String())
				}
				w.WriteHeader(http.StatusNoContent)
			})

			req := httptest.NewRequest("GET", "/any", nil)
			// inject chi URLParam
			rctx := chi.NewRouteContext()
			if tc.paramValue != "" {
				rctx.URLParams.Add("id", tc.paramValue)
			}
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			rec := httptest.NewRecorder()

			// call middleware
			handler := mw(next)
			handler.ServeHTTP(rec, req)

			if rec.Code != tc.wantStatus {
				t.Errorf("status = %d; want %d", rec.Code, tc.wantStatus)
			}
			if nextCalled != tc.expectNextCall {
				t.Errorf("nextCalled = %v; want %v", nextCalled, tc.expectNextCall)
			}
			if tc.expectNextCall {
				got := rec.Header().Get("X-ID")
				if got != tc.paramValue {
					t.Errorf("ID in context = %q; want %q", got, tc.paramValue)
				}
			}
		})
	}
}
