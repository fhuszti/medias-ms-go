package api

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/fhuszti/medias-ms-go/internal/logger"
)

type ErrorResponse struct {
	Error string `json:"error"`
}

func WriteError(w http.ResponseWriter, status int, msg string, err error) {
	ctx := context.Background()
	if err != nil {
		logger.Errorf(ctx, "❌  %s: %v", msg, err)
	} else {
		logger.Error(ctx, "❌  "+msg)
	}
	w.Header().Set("Cache-Control", "no-store, max-age=0, must-revalidate")
	RespondJSON(w, status, ErrorResponse{Error: msg})
}

func RespondJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		logger.Errorf(context.Background(), "❌  Failed to encode JSON response: %v", err)
	}
}

func RespondRawJSON(w http.ResponseWriter, status int, raw []byte) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if _, err := w.Write(raw); err != nil {
		logger.Errorf(context.Background(), "❌  Failed to write JSON payload: %v", err)
	}
}
