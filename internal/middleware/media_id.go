package middleware

import (
	"context"
	"fmt"
	"net/http"

	"github.com/fhuszti/medias-ms-go/internal/api_context"
	"github.com/fhuszti/medias-ms-go/internal/handler/api"
	msuuid "github.com/fhuszti/medias-ms-go/internal/uuid"
	"github.com/go-chi/chi/v5"
	guuid "github.com/google/uuid"
)

func WithMediaID() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			id := chi.URLParam(r, "id")
			if id == "" {
				api.WriteError(w, http.StatusBadRequest, "ID is required", nil)
				return
			}
			parsedID, err := guuid.Parse(id)
			if err != nil {
				api.WriteError(w, http.StatusBadRequest, fmt.Sprintf("ID %q is not a valid UUID", id), nil)
				return
			}

			// stash it in context and call the real handler
			ctx := context.WithValue(r.Context(), api_context.IDKey, msuuid.UUID(parsedID))
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
