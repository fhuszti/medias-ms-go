package api

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/fhuszti/medias-ms-go/internal/db"
	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"

	"github.com/go-chi/chi/v5"
)

func WithID() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			id := chi.URLParam(r, "id")
			if id == "" {
				WriteError(w, http.StatusBadRequest, "ID is required", nil)
				return
			}
			parsedID, err := uuid.Parse(id)
			if err != nil {
				WriteError(w, http.StatusBadRequest, fmt.Sprintf("ID %q is not a valid UUID", id), nil)
				return
			}

			// stash it in context and call the real handler
			ctx := context.WithValue(r.Context(), IDKey, db.UUID(parsedID))
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func WithJWTAuth(publicKeyPEM string) func(http.Handler) http.Handler {
	if publicKeyPEM == "" {
		return func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				next.ServeHTTP(w, r)
			})
		}
	}

	pubKey, err := jwt.ParseRSAPublicKeyFromPEM([]byte(publicKeyPEM))
	if err != nil {
		panic(fmt.Sprintf("invalid RSA public key: %v", err))
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			auth := r.Header.Get("Authorization")
			if !strings.HasPrefix(auth, "Bearer ") {
				WriteError(w, http.StatusUnauthorized, "missing bearer token", nil)
				return
			}
			tokenStr := strings.TrimPrefix(auth, "Bearer ")
			token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
				if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
					return nil, fmt.Errorf("unexpected signing method")
				}
				return pubKey, nil
			})
			if err != nil || !token.Valid {
				WriteError(w, http.StatusUnauthorized, "unauthorized", nil)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
