package media

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
)

func WithDestBucket(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		bucket := chi.URLParam(r, "destBucket")
		if bucket == "" {
			http.Error(w, "destination bucket is required", http.StatusBadRequest)
			return
		}
		// stash it in context and call the real handler
		ctx := context.WithValue(r.Context(), DestBucketKey, bucket)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
