package middleware

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
)

func TestAuthMiddleware_acceptsUserJWT(t *testing.T) {
	server, validator, key := testIAM(t)
	defer server.Close()
	withAuthValidator(t, validator)

	token := signTestJWT(t, key, map[string]any{
		"iss":         server.URL,
		"nbf":         time.Now().Add(-1 * time.Minute).Unix(),
		"iat":         time.Now().Add(-1 * time.Minute).Unix(),
		"exp":         time.Now().Add(1 * time.Hour).Unix(),
		"scope":       []string{"openid", "profile", "offline_access"},
		"amr":         []string{"external"},
		"client_id":   "tsid-client-demo",
		"sub":         "u-1",
		"auth_time":   time.Now().Add(-2 * time.Minute).Unix(),
		"idp":         "teamsystem",
		"name":        "Alice Demo",
		"given_name":  "Alice",
		"family_name": "Demo",
		"email":       "alice@example.com",
		"ncs_id":      "824dd6fd-d439-5e29-b8e6-1a9460664917",
		"sid":         "SESSION1",
		"jti":         "JWT1",
	})

	req := httptest.NewRequest(http.MethodGet, "/articles/1", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	c := echo.New().NewContext(req, rec)

	if err := AuthMiddleware(c); err != nil {
		t.Fatalf("AuthMiddleware: %v", err)
	}
	if rec.Code != 0 && rec.Code != http.StatusOK {
		t.Errorf("expected no early response, got %d", rec.Code)
	}
	ac := AuthContextFrom(c.Request().Context())
	if ac == nil || ac.Kind != AuthKindUser {
		t.Fatalf("expected AuthKindUser context, got %+v", ac)
	}
	if ac.UserID != "u-1" {
		t.Errorf("expected UserID=u-1, got %s", ac.UserID)
	}
	if ac.Email != "alice@example.com" {
		t.Errorf("expected email from JWT claims, got %s", ac.Email)
	}
	// TSID is derived from the OAuth `sub` claim, not `ncs_id` (which is a
	// Notification Center Service routing id, not an identity).
	if ac.TSID != "u-1" {
		t.Errorf("expected TSID mapped from sub, got %s", ac.TSID)
	}
	if !ac.HasScope("profile") {
		t.Errorf("expected profile scope, got %v", ac.Scopes)
	}
}

func TestAuthMiddleware_acceptsM2MJWT(t *testing.T) {
	server, validator, key := testIAM(t)
	defer server.Close()
	withAuthValidator(t, validator)

	token := signTestJWT(t, key, map[string]any{
		"iss":       server.URL,
		"nbf":       time.Now().Add(-1 * time.Minute).Unix(),
		"iat":       time.Now().Add(-1 * time.Minute).Unix(),
		"exp":       time.Now().Add(1 * time.Hour).Unix(),
		"scope":     []string{"platform.m2m"},
		"amr":       []string{"client_credentials"},
		"client_id": "svc-orders",
		"sub":       "svc-orders",
		"idp":       "teamsystem",
		"name":      "Orders Service",
		"sid":       "SESSION2",
		"jti":       "JWT2",
	})

	req := httptest.NewRequest(http.MethodGet, "/articles/1", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	c := echo.New().NewContext(req, rec)

	if err := AuthMiddleware(c); err != nil {
		t.Fatalf("AuthMiddleware: %v", err)
	}
	ac := AuthContextFrom(c.Request().Context())
	if ac == nil || ac.Kind != AuthKindM2M {
		t.Fatalf("expected AuthKindM2M context, got %+v", ac)
	}
	if ac.ServiceID != "svc-orders" {
		t.Errorf("expected ServiceID=svc-orders, got %s", ac.ServiceID)
	}
	if !ac.HasScope("platform.m2m") {
		t.Errorf("expected platform.m2m scope, got %v", ac.Scopes)
	}
}

func TestAuthMiddleware_rejectsMissingHeader(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := echo.New().NewContext(req, rec)

	_ = AuthMiddleware(c)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestAuthMiddleware_rejectsBadPrefix(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer notatoken")
	rec := httptest.NewRecorder()
	c := echo.New().NewContext(req, rec)

	_ = AuthMiddleware(c)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestAuthMiddleware_skipsConfiguredPaths(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	c := echo.New().NewContext(req, rec)

	if err := AuthMiddlewareWithSkip(c); err != nil {
		t.Fatalf("AuthMiddlewareWithSkip: %v", err)
	}
	if rec.Code != 0 && rec.Code != http.StatusOK {
		t.Errorf("expected no early response on /health, got %d", rec.Code)
	}
}

// The table below is the negative half of the Task 1 specification: each
// token is perfectly signed, but one claim is wrong, and the middleware
// must answer 401.
func TestAuthMiddleware_rejectsInvalidClaims(t *testing.T) {
	server, validator, key := testIAM(t)
	defer server.Close()
	withAuthValidator(t, validator)

	baseClaims := func() map[string]any {
		return map[string]any{
			"iss":       server.URL,
			"nbf":       time.Now().Add(-1 * time.Minute).Unix(),
			"iat":       time.Now().Add(-1 * time.Minute).Unix(),
			"exp":       time.Now().Add(1 * time.Hour).Unix(),
			"scope":     []string{"openid", "profile"},
			"amr":       []string{"external"},
			"client_id": "tsid-client-demo",
			"sub":       "u-1",
			"idp":       "teamsystem",
			"sid":       "SESSION1",
			"jti":       "JWT1",
		}
	}

	tests := []struct {
		name   string
		mutate func(claims map[string]any)
	}{
		{"an expired token is rejected", func(c map[string]any) {
			c["exp"] = time.Now().Add(-1 * time.Minute).Unix()
		}},
		{"a token from another issuer is rejected", func(c map[string]any) {
			c["iss"] = "http://evil-issuer.local"
		}},
		{"a token not valid yet is rejected", func(c map[string]any) {
			c["nbf"] = time.Now().Add(1 * time.Hour).Unix()
		}},
		{"a token without a subject is rejected", func(c map[string]any) {
			c["sub"] = ""
		}},
		{"a disallowed client_id is rejected", func(c map[string]any) {
			c["client_id"] = "evil-client"
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			claims := baseClaims()
			tt.mutate(claims)
			token := signTestJWT(t, key, claims)

			req := httptest.NewRequest(http.MethodGet, "/articles/1", nil)
			req.Header.Set("Authorization", "Bearer "+token)
			rec := httptest.NewRecorder()
			c := echo.New().NewContext(req, rec)

			_ = AuthMiddleware(c)
			if rec.Code != http.StatusUnauthorized {
				t.Errorf("expected 401, got %d", rec.Code)
			}
		})
	}
}

func testIAM(t *testing.T) (*httptest.Server, *JWTValidator, *rsa.PrivateKey) {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/.well-known/jwks.json" {
			http.NotFound(w, r)
			return
		}
		p := key.PublicKey
		_ = json.NewEncoder(w).Encode(map[string]any{
			"keys": []map[string]string{{
				"kty": "RSA",
				"use": "sig",
				"kid": "test-key",
				"alg": "RS512",
				"n":   b64url(p.N.Bytes()),
				"e":   b64url(big.NewInt(int64(p.E)).Bytes()),
			}},
		})
	}))
	validator := &JWTValidator{
		Issuer:           server.URL,
		JWKSURL:          server.URL + "/.well-known/jwks.json",
		AllowedClientIDs: map[string]bool{"tsid-client-demo": true, "svc-orders": true},
		HTTPClient:       server.Client(),
		Now:              time.Now,
		keys:             map[string]*rsa.PublicKey{},
	}
	return server, validator, key
}

func withAuthValidator(t *testing.T, validator TokenValidator) {
	t.Helper()
	old := authValidator
	authValidator = validator
	t.Cleanup(func() { authValidator = old })
}

func signTestJWT(t *testing.T, key *rsa.PrivateKey, claims map[string]any) string {
	t.Helper()
	header := map[string]any{"alg": "RS512", "kid": "test-key", "typ": "at+jwt"}
	headerJSON, _ := json.Marshal(header)
	claimsJSON, _ := json.Marshal(claims)
	signingInput := b64url(headerJSON) + "." + b64url(claimsJSON)
	h := crypto.SHA512.New()
	_, _ = h.Write([]byte(signingInput))
	sig, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA512, h.Sum(nil))
	if err != nil {
		t.Fatalf("SignPKCS1v15: %v", err)
	}
	return signingInput + "." + b64url(sig)
}

func b64url(bytes []byte) string {
	return base64.RawURLEncoding.EncodeToString(bytes)
}
