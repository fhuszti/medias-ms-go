package api

import (
	"encoding/json"
	"net/http"
)

func MethodNotAllowedHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusMethodNotAllowed)

		_ = json.NewEncoder(w).Encode("This method is not allowed")
	}
}
