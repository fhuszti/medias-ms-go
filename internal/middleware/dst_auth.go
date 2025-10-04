package middleware

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/fhuszti/medias-ms-go/internal/api_context"
	"github.com/fhuszti/medias-ms-go/internal/handler/api"
	"github.com/golang-jwt/jwt/v4"
)

// WithDSTAuth validates a short-lived Bearer JWT (DST only)
func WithDSTAuth(jwtPublicKeyPEM string) func(http.Handler) http.Handler {
	// Passthrough if no public key is provided
	if jwtPublicKeyPEM == "" {
		return func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				next.ServeHTTP(w, r)
			})
		}
	}

	pubKey, err := jwt.ParseRSAPublicKeyFromPEM([]byte(jwtPublicKeyPEM))
	if err != nil {
		panic(fmt.Sprintf("invalid Core RSA public key: %v", err))
	}

	parser := jwt.NewParser(
		jwt.WithValidMethods([]string{jwt.SigningMethodRS256.Name}),
	)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			auth := r.Header.Get("Authorization")
			if !strings.HasPrefix(auth, "Bearer ") {
				api.WriteError(w, http.StatusUnauthorized, "missing bearer token", nil)
				return
			}

			raw := strings.TrimPrefix(auth, "Bearer ")
			claims := jwt.MapClaims{}
			tok, err := parser.ParseWithClaims(raw, claims, func(t *jwt.Token) (interface{}, error) {
				if t.Method != jwt.SigningMethodRS256 {
					return nil, fmt.Errorf("unexpected signing method")
				}
				return pubKey, nil
			})
			if err != nil || !tok.Valid {
				api.WriteError(w, http.StatusUnauthorized, "unauthorized", nil)
				return
			}

			if !claims.VerifyIssuer("core", true) {
				api.WriteError(w, http.StatusUnauthorized, "bad issuer", nil)
				return
			}
			if !claims.VerifyAudience("medias", true) {
				api.WriteError(w, http.StatusUnauthorized, "bad audience", nil)
				return
			}
			if !claims.VerifyExpiresAt(time.Now().Unix(), true) {
				api.WriteError(w, http.StatusUnauthorized, "token expired", nil)
				return
			}
			if iat, ok := asInt64(claims["iat"]); ok && time.Unix(iat, 0).After(time.Now().Add(30*time.Second)) {
				api.WriteError(w, http.StatusUnauthorized, "invalid iat", nil)
				return
			}

			sub, _ := claims["sub"].(string)
			if sub == "" {
				api.WriteError(w, http.StatusUnauthorized, "missing sub", nil)
				return
			}
			roles := toStringSlice(claims["roles"])

			ctx := context.WithValue(r.Context(), api_context.AuthUserIDKey, sub)
			ctx = context.WithValue(ctx, api_context.AuthRolesKey, roles)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func asInt64(v any) (int64, bool) {
	switch x := v.(type) {
	case float64:
		return int64(x), true
	case json.Number:
		i, err := x.Int64()
		if err == nil {
			return i, true
		}
	}
	return 0, false
}

func toStringSlice(v any) []string {
	switch vv := v.(type) {
	case []string:
		return vv
	case []any:
		out := make([]string, 0, len(vv))
		for _, e := range vv {
			if s, ok := e.(string); ok {
				out = append(out, s)
			}
		}
		return out
	default:
		return nil
	}
}
