package handler

import (
	"encoding/json"
	"log"
	"net/http"
)

func WriteError(w http.ResponseWriter, status int, msg string, err error) {
	if err != nil {
		log.Printf("❌  %s: %v", msg, err)
	} else {
		log.Printf("❌  %s", msg)
	}
	http.Error(w, msg, status)
}

func RespondJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("❌  Failed to encode JSON response: %v", err)
	}
}

func RespondRawJSON(w http.ResponseWriter, status int, raw []byte) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if _, err := w.Write(raw); err != nil {
		log.Printf("❌  Failed to write JSON payload: %v", err)
	}
}
