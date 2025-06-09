package api

import (
	"encoding/json"
	"fmt"
	"github.com/fhuszti/medias-ms-go/internal/db"
	"github.com/fhuszti/medias-ms-go/internal/usecase/media"
	"github.com/fhuszti/medias-ms-go/internal/validation"
	"github.com/google/uuid"
	"log"
	"net/http"
)

type FinaliseUploadRequest struct {
	ID string `json:"id" validate:"required,uuid"`
}

func FinaliseUploadHandler(svc media.UploadFinaliser) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		destBucket, ok := BucketFromContext(r.Context())
		if !ok {
			WriteError(w, http.StatusBadRequest, "destination bucket is required", nil)
			return
		}

		var req FinaliseUploadRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			WriteError(w, http.StatusBadRequest, "invalid request payload", err)
			return
		}

		if errs := validation.ValidateStruct(req); errs != nil {
			errsJSON, err := validation.ErrorsToJson(errs)
			if err != nil {
				WriteError(w, http.StatusInternalServerError, "failed to encode validation errors", err)
				return
			}
			RespondRawJSON(w, http.StatusBadRequest, []byte(errsJSON))
			log.Printf("❌  Validation failed: %s", errsJSON)
			return
		}

		id, err := uuid.Parse(req.ID)
		if err != nil {
			WriteError(w, http.StatusBadRequest, "Invalid request", fmt.Errorf("invalid UUID: %w", err))
			return
		}

		input := media.FinaliseUploadInput{
			ID:         db.UUID(id),
			DestBucket: destBucket,
		}
		if err := svc.FinaliseUpload(r.Context(), input); err != nil {
			WriteError(w, http.StatusInternalServerError, fmt.Sprintf("could not finalise upload of media #%s", input.ID), err)
			return
		}

		w.WriteHeader(http.StatusNoContent)
		log.Printf("✅  Successfully finalised upload of media #%s", input.ID)
	}
}
