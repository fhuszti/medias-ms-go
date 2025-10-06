package api

import (
	"errors"
	"net/http"

	"github.com/fhuszti/medias-ms-go/internal/api_context"
	"github.com/fhuszti/medias-ms-go/internal/port"
	"github.com/fhuszti/medias-ms-go/internal/usecase/media"

	"github.com/fhuszti/medias-ms-go/internal/logger"
)

// DeleteMediaHandler deletes a media by ID.
func DeleteMediaHandler(svc port.MediaDeleter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := api_context.IDFromContext(r.Context())
		if !ok {
			WriteError(w, http.StatusBadRequest, "ID is required", nil)
			return
		}

		if err := svc.DeleteMedia(r.Context(), id); err != nil {
			if errors.Is(err, media.ErrObjectNotFound) {
				WriteError(w, http.StatusNotFound, "Media not found", nil)
				return
			}
			WriteError(w, http.StatusInternalServerError, "Failed to delete media", err)
			return
		}

		w.WriteHeader(http.StatusNoContent)
		logger.Infof(r.Context(), "✅  Successfully deleted media #%s", id)
	}
}
