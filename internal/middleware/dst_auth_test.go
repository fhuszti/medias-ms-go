package middleware

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/fhuszti/medias-ms-go/internal/api_context"
	"github.com/golang-jwt/jwt/v4"
)

func TestWithDSTAuth(t *testing.T) {
	privKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	pubDER, err := x509.MarshalPKIXPublicKey(&privKey.PublicKey)
	if err != nil {
		t.Fatalf("marshal public key: %v", err)
	}
	pubPEM := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubDER})

	middleware := WithDSTAuth(string(pubPEM))

	baseClaims := jwt.MapClaims{
		"iss":   "core",
		"aud":   "medias",
		"exp":   time.Now().Add(time.Minute).Unix(),
		"iat":   time.Now().Unix(),
		"sub":   "user-123",
		"roles": []any{"admin", "dst"},
	}

	tests := []struct {
		name           string
		modifyClaims   func(jwt.MapClaims) jwt.MapClaims
		tokenFactory   func(jwt.MapClaims) (string, error)
		authHeader     string
		wantStatus     int
		expectNextCall bool
	}{
		{
			name:           "missing header",
			authHeader:     "",
			wantStatus:     http.StatusUnauthorized,
			expectNextCall: false,
		},
		{
			name:           "wrong prefix",
			authHeader:     "Token abc",
			wantStatus:     http.StatusUnauthorized,
			expectNextCall: false,
		},
		{
			name:         "bad signature",
			modifyClaims: func(c jwt.MapClaims) jwt.MapClaims { return c },
			tokenFactory: func(claims jwt.MapClaims) (string, error) {
				otherKey, err := rsa.GenerateKey(rand.Reader, 1024)
				if err != nil {
					return "", err
				}
				return jwt.NewWithClaims(jwt.SigningMethodRS256, claims).SignedString(otherKey)
			},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:         "wrong method",
			modifyClaims: func(c jwt.MapClaims) jwt.MapClaims { return c },
			tokenFactory: func(claims jwt.MapClaims) (string, error) {
				return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte("secret"))
			},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name: "bad issuer",
			modifyClaims: func(c jwt.MapClaims) jwt.MapClaims {
				c = cloneClaims(c)
				c["iss"] = "other"
				return c
			},
			tokenFactory: func(claims jwt.MapClaims) (string, error) {
				return jwt.NewWithClaims(jwt.SigningMethodRS256, claims).SignedString(privKey)
			},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name: "bad audience",
			modifyClaims: func(c jwt.MapClaims) jwt.MapClaims {
				c = cloneClaims(c)
				c["aud"] = "other"
				return c
			},
			tokenFactory: func(claims jwt.MapClaims) (string, error) {
				return jwt.NewWithClaims(jwt.SigningMethodRS256, claims).SignedString(privKey)
			},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name: "expired",
			modifyClaims: func(c jwt.MapClaims) jwt.MapClaims {
				c = cloneClaims(c)
				c["exp"] = time.Now().Add(-time.Minute).Unix()
				return c
			},
			tokenFactory: func(claims jwt.MapClaims) (string, error) {
				return jwt.NewWithClaims(jwt.SigningMethodRS256, claims).SignedString(privKey)
			},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name: "future iat",
			modifyClaims: func(c jwt.MapClaims) jwt.MapClaims {
				c = cloneClaims(c)
				c["iat"] = time.Now().Add(time.Minute).Unix()
				return c
			},
			tokenFactory: func(claims jwt.MapClaims) (string, error) {
				return jwt.NewWithClaims(jwt.SigningMethodRS256, claims).SignedString(privKey)
			},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name: "missing sub",
			modifyClaims: func(c jwt.MapClaims) jwt.MapClaims {
				c = cloneClaims(c)
				delete(c, "sub")
				return c
			},
			tokenFactory: func(claims jwt.MapClaims) (string, error) {
				return jwt.NewWithClaims(jwt.SigningMethodRS256, claims).SignedString(privKey)
			},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:         "valid token",
			modifyClaims: func(c jwt.MapClaims) jwt.MapClaims { return cloneClaims(c) },
			tokenFactory: func(claims jwt.MapClaims) (string, error) {
				return jwt.NewWithClaims(jwt.SigningMethodRS256, claims).SignedString(privKey)
			},
			wantStatus:     http.StatusNoContent,
			expectNextCall: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			nextCalled := false
			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				nextCalled = true
				rawUserID := r.Context().Value(api_context.AuthUserIDKey)
				roles, _ := api_context.AuthRolesFromContext(r.Context())
				if sub, ok := rawUserID.(string); ok {
					w.Header().Set("X-User-ID", sub)
				}
				w.Header().Set("X-Roles", strings.Join(roles, ","))
				w.WriteHeader(http.StatusNoContent)
			})

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tc.authHeader != "" {
				req.Header.Set("Authorization", tc.authHeader)
			} else if tc.tokenFactory != nil {
				claims := cloneClaims(baseClaims)
				if tc.modifyClaims != nil {
					claims = tc.modifyClaims(claims)
				}
				token, err := tc.tokenFactory(claims)
				if err != nil {
					t.Fatalf("sign token: %v", err)
				}
				req.Header.Set("Authorization", "Bearer "+token)
			}

			rec := httptest.NewRecorder()

			handler := middleware(next)
			handler.ServeHTTP(rec, req)

			if rec.Code != tc.wantStatus {
				t.Fatalf("status = %d; want %d", rec.Code, tc.wantStatus)
			}
			if nextCalled != tc.expectNextCall {
				t.Fatalf("nextCalled = %v; want %v", nextCalled, tc.expectNextCall)
			}
			if tc.expectNextCall {
				if got := rec.Header().Get("X-User-ID"); got != baseClaims["sub"].(string) {
					t.Fatalf("user id = %q; want %q", got, baseClaims["sub"])
				}
				if got := rec.Header().Get("X-Roles"); got != "admin,dst" {
					t.Fatalf("roles = %q; want %q", got, "admin,dst")
				}
			}
		})
	}
}

func cloneClaims(src jwt.MapClaims) jwt.MapClaims {
	dst := make(jwt.MapClaims, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}
