package middleware

import "warehouse.local/core/policies"

// The authorization decision of the Warehouse BC.
//
// Authentication (auth.go) answered "who are you?". This file asks the next
// question, "may YOU do THIS?", but no longer answers it: the rules live in
// policies/warehouse.rego. Can only translates the AuthContext into the
// policy's input document and relays the verdict.
//
// Note the family resemblance with the auth allowlist: the bouncer checked
// which APPLICATION asked for the token; the policy checks what THIS
// identity may do. Same shape (a list and a default), one level up, and the
// list and the default now live in a policy file, not in Go.

// Actions the Warehouse BC knows about.
const (
	ActionArticleRead   = "article:read"
	ActionArticleCreate = "article:create"
)

// Can is the authorization decision: may this caller perform this action?
// Given code: the decision itself is policies/warehouse.rego, Part 2 territory.
func Can(ac *AuthContext, action string) bool {
	input := map[string]any{"action": action}
	if ac != nil {
		input["principal"] = map[string]any{
			"kind":       principalKind(ac.Kind),
			"email":      ac.Email,
			"service_id": ac.ServiceID,
		}
	}
	return policies.Allow(input)
}

func principalKind(k AuthKind) string {
	switch k {
	case AuthKindUser:
		return "user"
	case AuthKindM2M:
		return "m2m"
	default:
		return "unknown"
	}
}
