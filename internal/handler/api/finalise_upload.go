package api

import (
	"encoding/json"
	"fmt"
	"github.com/fhuszti/medias-ms-go/internal/usecase/media"
	"github.com/fhuszti/medias-ms-go/internal/validation"
	"log"
	"net/http"
)

type FinaliseUploadRequest struct {
	DestBucket string `json:"destBucket" validate:"required"`
}

func FinaliseUploadHandler(svc media.UploadFinaliser, allowed []string) http.HandlerFunc {
	allowedSet := make(map[string]struct{}, len(allowed))
	for _, b := range allowed {
		allowedSet[b] = struct{}{}
	}

	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := IDFromContext(r.Context())
		if !ok {
			WriteError(w, http.StatusBadRequest, "ID is required", nil)
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

		if _, ok := allowedSet[req.DestBucket]; !ok {
			WriteError(w, http.StatusBadRequest, fmt.Sprintf("destination bucket %q does not exist", req.DestBucket), nil)
			return
		}

		input := media.FinaliseUploadInput{
			ID:         id,
			DestBucket: req.DestBucket,
		}
		if err := svc.FinaliseUpload(r.Context(), input); err != nil {
			WriteError(w, http.StatusInternalServerError, fmt.Sprintf("could not finalise upload of media #%s", input.ID), err)
			return
		}

		w.WriteHeader(http.StatusNoContent)
		log.Printf("✅  Successfully finalised upload of media #%s", input.ID)
	}
}
