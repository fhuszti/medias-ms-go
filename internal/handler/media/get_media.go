package media

import (
	"encoding/json"
	"fmt"
	"github.com/fhuszti/medias-ms-go/internal/db"
	"github.com/fhuszti/medias-ms-go/internal/handler"
	"github.com/fhuszti/medias-ms-go/internal/usecase/media"
	"github.com/fhuszti/medias-ms-go/internal/validation"
	"github.com/google/uuid"
	"log"
	"net/http"
)

type GetMediaRequest struct {
	ID string `json:"id" validate:"required,uuid"`
}

func GetMediaHandler(svc media.Getter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req GetMediaRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			handler.WriteError(w, http.StatusBadRequest, "Invalid request", fmt.Errorf("invalid JSON: %w", err))
			return
		}

		if errs := validation.ValidateStruct(req); errs != nil {
			errsJSON, err := validation.ErrorsToJson(errs)
			if err != nil {
				handler.WriteError(w, http.StatusInternalServerError, "Validation error (could not encode details)", fmt.Errorf("encoding validation errors: %w", err))
				return
			}

			// return the validation errors payload directly
			handler.RespondRawJSON(w, http.StatusBadRequest, []byte(errsJSON))
			log.Printf("❌  Validation failed: %s", errsJSON)
			return
		}

		id, err := uuid.Parse(req.ID)
		if err != nil {
			handler.WriteError(w, http.StatusBadRequest, "Invalid request", fmt.Errorf("invalid UUID: %w", err))
			return
		}

		in := media.GetMediaInput{ID: db.UUID(id)}
		out, err := svc.GetMedia(r.Context(), in)
		if err != nil {
			handler.WriteError(w, http.StatusInternalServerError, "Could not get media details", err)
			return
		}

		if out.Optimised {
			// public cache for 20 minutes
			w.Header().Set("Cache-Control", "public, max-age=1200")
		} else {
			// no caching when still unoptimised
			w.Header().Set("Cache-Control", "no-store, max-age=0, must-revalidate")
		}

		handler.RespondJSON(w, http.StatusCreated, out)
		log.Printf("✅  Successfully returned details for media #%s", in.ID)
	}
}
