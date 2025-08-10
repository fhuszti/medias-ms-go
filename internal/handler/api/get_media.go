package api

import (
	"errors"
	"log"
	"net/http"

	"github.com/fhuszti/medias-ms-go/internal/port"
	"github.com/fhuszti/medias-ms-go/internal/usecase/media"
)

func GetMediaHandler(renderer port.HTTPRenderer, svc port.MediaGetter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := IDFromContext(r.Context())
		if !ok {
			WriteError(w, http.StatusBadRequest, "ID is required", nil)
			return
		}

		raw, etag, err := renderer.RenderGetMedia(r.Context(), svc, id)
		if err != nil {
			if errors.Is(err, media.ErrObjectNotFound) {
				WriteError(w, http.StatusNotFound, "Media not found", nil)
				return
			}
			WriteError(w, http.StatusInternalServerError, "Could not get media details", err)
			return
		}

		w.Header().Set("ETag", etag)
		w.Header().Set("Cache-Control", "public, max-age=300")
		if match := r.Header.Get("If-None-Match"); match == etag {
			w.WriteHeader(http.StatusNotModified)
			log.Printf("✅  Returning cached media #%s", id)
			return
		}

		RespondRawJSON(w, http.StatusOK, raw)
		log.Printf("✅  Successfully returned details for media #%s", id)
	}
}
