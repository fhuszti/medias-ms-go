package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"hash/crc32"
	"log"
	"net/http"

	"github.com/fhuszti/medias-ms-go/internal/port"
	media "github.com/fhuszti/medias-ms-go/internal/usecase/media"
)

func GetMediaHandler(svc port.MediaGetter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := IDFromContext(r.Context())
		if !ok {
			WriteError(w, http.StatusBadRequest, "ID is required", nil)
			return
		}

		in := port.GetMediaInput{ID: id}
		out, err := svc.GetMedia(r.Context(), in)
		if err != nil {
			if errors.Is(err, media.ErrObjectNotFound) {
				WriteError(w, http.StatusNotFound, "Media not found", nil)
				return
			}
			WriteError(w, http.StatusInternalServerError, "Could not get media details", err)
			return
		}

		raw, err := json.Marshal(out)
		if err != nil {
			WriteError(w, http.StatusInternalServerError, "Could not encode response", err)
			return
		}
		etag := fmt.Sprintf("\"%08x\"", crc32.ChecksumIEEE(raw))
		w.Header().Set("ETag", etag)
		w.Header().Set("Cache-Control", "max-age=0")
		if match := r.Header.Get("If-None-Match"); match == etag {
			w.WriteHeader(http.StatusNotModified)
			return
		}

		RespondRawJSON(w, http.StatusOK, raw)
		log.Printf("✅  Successfully returned details for media #%s", in.ID)
	}
}
