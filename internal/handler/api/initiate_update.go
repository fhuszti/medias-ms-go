package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/fhuszti/medias-ms-go/internal/api_context"
	"github.com/fhuszti/medias-ms-go/internal/port"
	"github.com/fhuszti/medias-ms-go/internal/usecase/media"
	"github.com/fhuszti/medias-ms-go/internal/validation"

	"github.com/fhuszti/medias-ms-go/internal/logger"
)

type InitiateUpdateRequest struct {
	Name string `json:"name" validate:"required,max=80"`
}

func InitiateUpdateHandler(svc port.UpdateInitiator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := api_context.IDFromContext(r.Context())
		if !ok {
			WriteError(w, http.StatusBadRequest, "ID is required", nil)
			return
		}

		var req InitiateUpdateRequest
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

			RespondRawJSON(w, http.StatusBadRequest, []byte(errsJSON))
			logger.Warnf(r.Context(), "❌  Validation failed: %s", errsJSON)
			return
		}

		in := port.InitiateUpdateInput{
			ID:   id,
			Name: req.Name,
		}
		out, err := svc.InitiateUpdate(r.Context(), in)
		if err != nil {
			if errors.Is(err, media.ErrObjectNotFound) {
				WriteError(w, http.StatusNotFound, "Media not found", nil)
				return
			}
			WriteError(w, http.StatusInternalServerError, "Could not initiate update", err)
			return
		}

		RespondJSON(w, http.StatusCreated, out)
		logger.Infof(r.Context(), "✅  Successfully initiated update for media #%s", out.ID)
	}
}
