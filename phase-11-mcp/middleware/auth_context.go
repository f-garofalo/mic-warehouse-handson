package middleware

import "context"

// AuthKind enumerates the supported principal types per ADR-015.
type AuthKind int

const (
	// AuthKindUnknown is the zero value; treat as unauthenticated.
	AuthKindUnknown AuthKind = iota
	// AuthKindUser indicates a user JWT (human-driven traffic).
	AuthKindUser
	// AuthKindM2M indicates a service-to-service token (client_credentials grant).
	AuthKindM2M
)

// AuthContext carries the authenticated principal plus correlation keys
// (TSID, WorkspaceID) that downstream layers propagate to logs and traces
// per ADR-017.
type AuthContext struct {
	Kind AuthKind

	// User-only fields.
	UserID      string
	Email       string
	WorkspaceID string

	// M2M-only fields.
	ServiceID string

	// TSID-like profile fields surfaced by the IAM mock response.
	Name             string
	GivenName        string
	FamilyName       string
	NCSID            string
	SessionID        string
	IdentityProvider string
	ClientID         string

	// Common fields.
	Scopes []string
	TSID   string
}

// IsUser reports whether the principal is a human user.
func (a AuthContext) IsUser() bool { return a.Kind == AuthKindUser }

// IsM2M reports whether the principal is a service.
func (a AuthContext) IsM2M() bool { return a.Kind == AuthKindM2M }

// Subject returns the user id or the service id depending on Kind.
// Suitable for log lines and audit fields. Empty for AuthKindUnknown.
func (a AuthContext) Subject() string {
	switch a.Kind {
	case AuthKindUser:
		return a.UserID
	case AuthKindM2M:
		return a.ServiceID
	default:
		return ""
	}
}

// HasScope reports whether the auth context carries the requested OAuth scope.
// Exact-match only; no wildcards. Domain permissions are decided later by policies.
func (a AuthContext) HasScope(want string) bool {
	for _, s := range a.Scopes {
		if s == want {
			return true
		}
	}
	return false
}

type authContextKey struct{}

// WithAuthContext returns a derived context carrying the AuthContext.
func WithAuthContext(parent context.Context, ac *AuthContext) context.Context {
	return context.WithValue(parent, authContextKey{}, ac)
}

// AuthContextFrom retrieves the AuthContext from a context, or nil if absent.
func AuthContextFrom(ctx context.Context) *AuthContext {
	if v := ctx.Value(authContextKey{}); v != nil {
		if ac, ok := v.(*AuthContext); ok {
			return ac
		}
	}
	return nil
}
