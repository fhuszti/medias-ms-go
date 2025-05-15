package media

import (
	"encoding/json"
	"fmt"
	"github.com/fhuszti/medias-ms-go/internal/db"
	"github.com/fhuszti/medias-ms-go/internal/usecase/media"
	"github.com/fhuszti/medias-ms-go/internal/validation"
	"log"
	"net/http"
)

type FinaliseUploadRequest struct {
	ID db.UUID `json:"id" validate:"required,uuid"`
}

func FinaliseUploadHandler(svc media.UploadFinaliser) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		destBucket, ok := BucketFromContext(r.Context())
		if !ok {
			writeError(w, http.StatusBadRequest, "destination bucket is required", nil)
			return
		}

		var req FinaliseUploadRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request payload", err)
			return
		}

		if errs := validation.ValidateStruct(req); errs != nil {
			errsJSON, err := validation.ErrorsToJson(errs)
			if err != nil {
				writeError(w, http.StatusInternalServerError, "failed to encode validation errors", err)
				return
			}
			respondRawJSON(w, http.StatusBadRequest, []byte(errsJSON))
			log.Printf("❌  Validation failed: %s", errsJSON)
			return
		}

		input := media.FinaliseUploadInput{
			ID:         req.ID,
			DestBucket: destBucket,
		}
		output, err := svc.FinaliseUpload(r.Context(), input)
		if err != nil {
			writeError(w, http.StatusInternalServerError, fmt.Sprintf("could not finalise upload of media #%s", input.ID), err)
			return
		}

		respondJSON(w, http.StatusOK, output)
		log.Printf("✅  Successfully finalised upload of media #%s", input.ID)
	}
}
