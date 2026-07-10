package middleware

import (
	"context"
	"crypto"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/labstack/echo/v4"
)

type TokenValidator interface {
	Validate(ctx context.Context, token string) (*AuthContext, error)
}

var authValidator TokenValidator = NewJWTValidatorFromEnv()

// AuthMiddleware validates the Authorization header and injects an
// *AuthContext into the request context. In Phase 07 the token is a TSID-like
// access token: RS512 JWT, typ=at+jwt, validated through the IAM mock JWKS.
func AuthMiddleware(c echo.Context) error {
	authHeader := c.Request().Header.Get("Authorization")
	if authHeader == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "missing authorization header",
		})
	}
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || parts[0] != "Bearer" {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "invalid authorization header format",
		})
	}

	ac, err := authValidator.Validate(c.Request().Context(), parts[1])
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "invalid token",
		})
	}

	req := c.Request().WithContext(WithAuthContext(c.Request().Context(), ac))
	c.SetRequest(req)
	return nil
}

// SkipAuthPath returns true if path should skip authentication.
func SkipAuthPath(path string) bool {
	skipPaths := map[string]bool{
		"/health":      true,
		"/metrics":     true,
		"/api/v1/auth": true,
	}
	return skipPaths[path]
}

// AuthMiddlewareWithSkip wraps AuthMiddleware with path skipping.
func AuthMiddlewareWithSkip(c echo.Context) error {
	if SkipAuthPath(c.Request().URL.Path) {
		return nil
	}
	return AuthMiddleware(c)
}

type JWTValidator struct {
	Issuer           string
	JWKSURL          string
	AllowedClientIDs map[string]bool
	HTTPClient       *http.Client
	Now              func() time.Time

	mu   sync.RWMutex
	keys map[string]*rsa.PublicKey
}

func NewJWTValidatorFromEnv() *JWTValidator {
	return &JWTValidator{
		Issuer:           envOr("IAM_ISSUER", "http://iam-mock:9001"),
		JWKSURL:          envOr("IAM_JWKS_URL", "http://iam-mock:9001/.well-known/jwks.json"),
		AllowedClientIDs: parseCSVSet(os.Getenv("IAM_ALLOWED_CLIENT_IDS")),
		HTTPClient:       &http.Client{Timeout: 5 * time.Second},
		Now:              time.Now,
		keys:             map[string]*rsa.PublicKey{},
	}
}

func (v *JWTValidator) Validate(ctx context.Context, token string) (*AuthContext, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, errors.New("jwt: expected three parts")
	}

	var header jwtHeader
	if err := decodeJSONPart(parts[0], &header); err != nil {
		return nil, fmt.Errorf("jwt header: %w", err)
	}
	if header.Alg != "RS512" {
		return nil, fmt.Errorf("jwt: unsupported alg %q", header.Alg)
	}
	if header.KID == "" {
		return nil, errors.New("jwt: missing kid")
	}

	key, err := v.key(ctx, header.KID)
	if err != nil {
		return nil, err
	}
	if err := verifyRS512(parts[0]+"."+parts[1], parts[2], key); err != nil {
		return nil, err
	}

	var claims tsidClaims
	if err := decodeJSONPart(parts[1], &claims); err != nil {
		return nil, fmt.Errorf("jwt claims: %w", err)
	}
	if err := v.validateClaims(claims); err != nil {
		return nil, err
	}
	return claims.AuthContext(), nil
}

func (v *JWTValidator) validateClaims(c tsidClaims) error {
	now := v.Now().Unix()
	if c.Issuer != v.Issuer {
		return fmt.Errorf("jwt: issuer mismatch")
	}
	if c.ExpiresAt <= now {
		return errors.New("jwt: expired")
	}
	if c.NotBefore > now {
		return errors.New("jwt: not valid yet")
	}
	if c.Subject == "" {
		return errors.New("jwt: missing sub")
	}
	if c.ClientID == "" {
		return errors.New("jwt: missing client_id")
	}
	if len(v.AllowedClientIDs) > 0 && !v.AllowedClientIDs[c.ClientID] {
		return errors.New("jwt: client_id not allowed")
	}
	return nil
}

func (v *JWTValidator) key(ctx context.Context, kid string) (*rsa.PublicKey, error) {
	v.mu.RLock()
	key := v.keys[kid]
	v.mu.RUnlock()
	if key != nil {
		return key, nil
	}
	if err := v.refreshKeys(ctx); err != nil {
		return nil, err
	}
	v.mu.RLock()
	defer v.mu.RUnlock()
	key = v.keys[kid]
	if key == nil {
		return nil, fmt.Errorf("jwks: key %q not found", kid)
	}
	return key, nil
}

func (v *JWTValidator) refreshKeys(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, v.JWKSURL, nil)
	if err != nil {
		return err
	}
	client := v.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("jwks fetch: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("jwks fetch: status %d", resp.StatusCode)
	}
	var jwks jwksDocument
	if err := json.NewDecoder(resp.Body).Decode(&jwks); err != nil {
		return fmt.Errorf("jwks decode: %w", err)
	}
	keys := map[string]*rsa.PublicKey{}
	for _, k := range jwks.Keys {
		if k.Kty != "RSA" || k.Kid == "" {
			continue
		}
		n, err := decodeBase64URL(k.N)
		if err != nil {
			return err
		}
		e, err := decodeBase64URL(k.E)
		if err != nil {
			return err
		}
		keys[k.Kid] = &rsa.PublicKey{N: new(big.Int).SetBytes(n), E: int(new(big.Int).SetBytes(e).Int64())}
	}
	v.mu.Lock()
	v.keys = keys
	v.mu.Unlock()
	return nil
}

type jwtHeader struct {
	Alg string `json:"alg"`
	KID string `json:"kid"`
	Typ string `json:"typ"`
}

type tsidClaims struct {
	Issuer           string   `json:"iss"`
	NotBefore        int64    `json:"nbf"`
	IssuedAt         int64    `json:"iat"`
	ExpiresAt        int64    `json:"exp"`
	Scopes           []string `json:"scope"`
	AMR              []string `json:"amr"`
	ClientID         string   `json:"client_id"`
	Subject          string   `json:"sub"`
	AuthTime         int64    `json:"auth_time"`
	IdentityProvider string   `json:"idp"`
	Name             string   `json:"name"`
	GivenName        string   `json:"given_name"`
	FamilyName       string   `json:"family_name"`
	Email            string   `json:"email"`
	NCSID            string   `json:"ncs_id"`
	SessionID        string   `json:"sid"`
	JTI              string   `json:"jti"`
	Locale           string   `json:"locale"`
}

func (c tsidClaims) AuthContext() *AuthContext {
	kind := AuthKindUser
	serviceID := ""
	userID := c.Subject
	if contains(c.AMR, "client_credentials") || strings.HasPrefix(c.Subject, "svc-") {
		kind = AuthKindM2M
		serviceID = c.Subject
		userID = ""
	}
	return &AuthContext{
		Kind:             kind,
		UserID:           userID,
		Email:            c.Email,
		ServiceID:        serviceID,
		Scopes:           c.Scopes,
		// TSID is the canonical user identity, derived from the OAuth `sub`
		// claim (OIDC requires `sub` to be present and stable). `ncs_id` is
		// a Notification Center Service routing identifier — used to address
		// the recipient when delivering notifications, NOT an identity claim.
		// Preferring `ncs_id` over `sub` would tie our audit trail to the
		// notification subsystem rather than to OAuth, and would break the
		// moment a user is configured to receive notifications via a
		// different channel (e.g. shared mailbox). We fall back to NCSID
		// only as defence-in-depth: with a valid JWT `sub` is always present.
		TSID:             firstNonEmpty(c.Subject, c.NCSID),
		NCSID:            c.NCSID,
		SessionID:        c.SessionID,
		IdentityProvider: c.IdentityProvider,
		ClientID:         c.ClientID,
		Name:             c.Name,
		GivenName:        c.GivenName,
		FamilyName:       c.FamilyName,
	}
}

type jwksDocument struct {
	Keys []jwkKey `json:"keys"`
}

type jwkKey struct {
	Kty string `json:"kty"`
	Use string `json:"use"`
	Kid string `json:"kid"`
	Alg string `json:"alg"`
	N   string `json:"n"`
	E   string `json:"e"`
}

func verifyRS512(signingInput, sigPart string, key *rsa.PublicKey) error {
	sig, err := decodeBase64URL(sigPart)
	if err != nil {
		return err
	}
	h := crypto.SHA512.New()
	_, _ = h.Write([]byte(signingInput))
	if err := rsa.VerifyPKCS1v15(key, crypto.SHA512, h.Sum(nil), sig); err != nil {
		return fmt.Errorf("jwt: invalid signature")
	}
	return nil
}

func decodeJSONPart(part string, out any) error {
	b, err := decodeBase64URL(part)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, out)
}

func decodeBase64URL(s string) ([]byte, error) {
	return base64.RawURLEncoding.DecodeString(s)
}

func parseCSVSet(s string) map[string]bool {
	out := map[string]bool{}
	for _, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(part)
		if part != "" {
			out[part] = true
		}
	}
	return out
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func contains(values []string, want string) bool {
	for _, v := range values {
		if v == want {
			return true
		}
	}
	return false
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}
