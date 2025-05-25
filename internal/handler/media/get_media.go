package media

import (
	"errors"
	"github.com/fhuszti/medias-ms-go/internal/handler"
	"github.com/fhuszti/medias-ms-go/internal/usecase/media"
	"log"
	"net/http"
)

func GetMediaHandler(svc media.Getter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := IDFromContext(r.Context())
		if !ok {
			handler.WriteError(w, http.StatusBadRequest, "ID is required", nil)
			return
		}

		in := media.GetMediaInput{ID: id}
		out, err := svc.GetMedia(r.Context(), in)
		if err != nil {
			if errors.Is(err, media.ErrObjectNotFound) {
				handler.WriteError(w, http.StatusNotFound, "Media not found", nil)
				return
			}
			handler.WriteError(w, http.StatusInternalServerError, "Could not get media details", err)
			return
		}

		isImage := media.IsImage(out.Metadata.MimeType)
		hasVariants := len(out.Variants) > 0
		isResized := !isImage || (isImage && hasVariants)
		isBytesOptimised := out.Optimised
		shouldCache := isBytesOptimised && isResized
		if shouldCache {
			// public cache for 20 minutes
			w.Header().Set("Cache-Control", "public, max-age=1200")
		} else {
			// no caching when still unoptimised
			w.Header().Set("Cache-Control", "no-store, max-age=0, must-revalidate")
		}

		handler.RespondJSON(w, http.StatusOK, out)
		log.Printf("✅  Successfully returned details for media #%s", in.ID)
	}
}
