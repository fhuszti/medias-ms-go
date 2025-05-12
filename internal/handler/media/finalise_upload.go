package media

import (
	"encoding/json"
	"fmt"
	"github.com/fhuszti/medias-ms-go/internal/db"
	"github.com/fhuszti/medias-ms-go/internal/usecase/media"
	"github.com/fhuszti/medias-ms-go/internal/validation"
	"github.com/go-chi/chi/v5"
	"log"
	"net/http"
)

type FinaliseUploadRequest struct {
	ID db.UUID `json:"id" validate:"required,uuid"`
}

func FinaliseUploadHandler(svc media.UploadFinaliser) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		destBucket := chi.URLParam(r, "destBucket")
		if destBucket == "" {
			errStr := "❌  Invalid request: a destination bucket is necessary for this operation"
			log.Print(errStr)
			http.Error(w, errStr, http.StatusBadRequest)
			return
		}

		var req FinaliseUploadRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			errStr := fmt.Sprintf("❌  Invalid request: %s", err.Error())
			log.Print(errStr)
			http.Error(w, errStr, http.StatusBadRequest)
			return
		}

		if errs := validation.ValidateStruct(req); errs != nil {
			errsJson, err := validation.ErrorsToJson(errs)
			if err != nil {
				errStr := fmt.Sprintf("❌  Could not encode validation errors: %s", err.Error())
				log.Print(errStr)
				http.Error(w, errStr, http.StatusInternalServerError)
			}

			http.Error(w, errsJson, http.StatusBadRequest)
			return
		}

		in := media.FinaliseUploadInput{
			ID:         req.ID,
			DestBucket: destBucket,
		}

		output, err := svc.FinaliseUpload(r.Context(), in)
		if err != nil {
			errStr := fmt.Sprintf("❌  Could not finalise upload of media #%s: %s", in.ID, err.Error())
			log.Print(errStr)
			http.Error(w, errStr, http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		err = json.NewEncoder(w).Encode(output)
		if err != nil {
			errStr := fmt.Sprintf("❌  Could not encode media entity following upload completion: %s", err.Error())
			log.Print(errStr)
			http.Error(w, errStr, http.StatusInternalServerError)
			return
		}

		log.Printf("✅  Successfully finalised upload of media #%s", in.ID)
	}
}
