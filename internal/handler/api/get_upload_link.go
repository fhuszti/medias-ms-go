package api

import (
	"encoding/json"
	"fmt"
	"github.com/fhuszti/medias-ms-go/internal/port"
	"github.com/fhuszti/medias-ms-go/internal/validation"
	"log"
	"net/http"
)

type GenerateUploadLinkRequest struct {
	Name string `json:"name" validate:"required,max=80"`
}

func GenerateUploadLinkHandler(svc port.UploadLinkGenerator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req GenerateUploadLinkRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			WriteError(w, http.StatusBadRequest, "Invalid request", fmt.Errorf("invalid JSON: %w", err))
			return
		}

		if errs := validation.ValidateStruct(req); errs != nil {
			errsJSON, err := validation.ErrorsToJson(errs)
			if err != nil {
				WriteError(w, http.StatusInternalServerError, "Validation error (could not encode details)", fmt.Errorf("encoding validation errors: %w", err))
				return
			}

			// return the validation errors payload directly
			RespondRawJSON(w, http.StatusBadRequest, []byte(errsJSON))
			log.Printf("❌  Validation failed: %s", errsJSON)
			return
		}

		in := port.GenerateUploadLinkInput(req)
		out, err := svc.GenerateUploadLink(r.Context(), in)
		if err != nil {
			WriteError(w, http.StatusInternalServerError, "Could not generate upload link", err)
			return
		}

		RespondJSON(w, http.StatusCreated, out)
		log.Printf("✅  Successfully generated upload link for media #%s", out.ID)
	}
}
