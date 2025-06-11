package api

import (
	"errors"
	"github.com/fhuszti/medias-ms-go/internal/usecase/media"
	"log"
	"net/http"
)

// DeleteMediaHandler deletes a media by ID.
func DeleteMediaHandler(svc media.Deleter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := IDFromContext(r.Context())
		if !ok {
			WriteError(w, http.StatusBadRequest, "ID is required", nil)
			return
		}

		in := media.DeleteMediaInput{ID: id}
		if err := svc.DeleteMedia(r.Context(), in); err != nil {
			if errors.Is(err, media.ErrObjectNotFound) {
				WriteError(w, http.StatusNotFound, "Media not found", nil)
				return
			}
			WriteError(w, http.StatusInternalServerError, "Failed to delete media", err)
			return
		}

		w.WriteHeader(http.StatusNoContent)
		log.Printf("✅  Successfully deleted media #%s", id)
	}
}
