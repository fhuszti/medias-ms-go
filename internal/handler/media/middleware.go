package media

import (
	"context"
	"fmt"
	"github.com/fhuszti/medias-ms-go/internal/db"
	"github.com/fhuszti/medias-ms-go/internal/handler"
	"github.com/google/uuid"
	"net/http"

	"github.com/go-chi/chi/v5"
)

func WithDestBucket(allowed []string) func(http.Handler) http.Handler {
	// turn the slice into a map for fast lookup
	m := make(map[string]struct{}, len(allowed))
	for _, b := range allowed {
		m[b] = struct{}{}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			bucket := chi.URLParam(r, "destBucket")
			if bucket == "" {
				handler.WriteError(w, http.StatusBadRequest, "destination bucket is required", nil)
				return
			}
			if _, ok := m[bucket]; !ok {
				handler.WriteError(w, http.StatusBadRequest, fmt.Sprintf("destination bucket %q does not exist", bucket), nil)
				return
			}

			// stash it in context and call the real handler
			ctx := context.WithValue(r.Context(), DestBucketKey, bucket)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func WithID() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			id := chi.URLParam(r, "id")
			if id == "" {
				handler.WriteError(w, http.StatusBadRequest, "ID is required", nil)
				return
			}
			parsedID, err := uuid.Parse(id)
			if err != nil {
				handler.WriteError(w, http.StatusBadRequest, fmt.Sprintf("ID %q is not a valid UUID", id), nil)
				return
			}

			// stash it in context and call the real handler
			ctx := context.WithValue(r.Context(), IDKey, db.UUID(parsedID))
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
