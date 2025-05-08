package media

import (
	"encoding/json"
	"fmt"
	"github.com/fhuszti/medias-ms-go/internal/usecase/media"
	"github.com/fhuszti/medias-ms-go/internal/validation"
	"net/http"
)

type GenerateUploadLinkRequest struct {
	Name string `json:"name" validate:"required,max=80"`
	Type string `json:"type" validate:"required,mimetype"`
}

func GenerateUploadLinkHandler(svc media.UploadLinkGenerator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req GenerateUploadLinkRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, fmt.Sprintf("Invalid request: %s", err.Error()), http.StatusBadRequest)
			return
		}

		if errs := validation.ValidateStruct(req); errs != nil {
			errsJson, err := validation.ErrorsToJson(errs)
			if err != nil {
				http.Error(w, "Could not encode validation errors: "+err.Error(), http.StatusInternalServerError)
			}

			http.Error(w, errsJson, http.StatusBadRequest)
			return
		}

		in := media.GenerateUploadLinkInput(req)

		presignedUrl, err := svc.GenerateUploadLink(r.Context(), in)
		if err != nil {
			http.Error(w, fmt.Sprintf("Could not generate presigned URL for upload: %s", err.Error()), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)

		err = json.NewEncoder(w).Encode(presignedUrl)
		if err != nil {
			http.Error(w, "Could not encode newly generated presigned URL for upload: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}
}
