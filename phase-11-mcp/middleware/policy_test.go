package middleware

import "testing"

// The table below is the specification of the policy (written in Phase 07,
// extended in Phase 11 with the MCP server's machine identity).
func TestCan(t *testing.T) {
	alice := &AuthContext{Kind: AuthKindUser, UserID: "049823c2-27f9-4507-a5c7-432c3efef275", Email: "alice@example.com"}
	bob := &AuthContext{Kind: AuthKindUser, UserID: "37dd6fad-7a82-4662-85bb-9fd7f9db83b1", Email: "bob@example.com"}
	orders := &AuthContext{Kind: AuthKindM2M, ServiceID: "svc-orders"}
	agent := &AuthContext{Kind: AuthKindM2M, ServiceID: "svc-warehouse-agent"}

	tests := []struct {
		name   string
		ac     *AuthContext
		action string
		want   bool
	}{
		{"no identity is always denied", nil, ActionArticleRead, false},
		{"every authenticated caller may read (Alice)", alice, ActionArticleRead, true},
		{"every authenticated caller may read (Bob)", bob, ActionArticleRead, true},
		{"the warehouse manager may create", alice, ActionArticleCreate, true},
		{"Bob may read but not create", bob, ActionArticleCreate, false},
		{"the orders service may create (M2M)", orders, ActionArticleCreate, true},
		{"the warehouse agent may create (M2M)", agent, ActionArticleCreate, true},
		{"unknown actions are denied by default", alice, "article:delete", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Can(tt.ac, tt.action); got != tt.want {
				t.Errorf("Can(%v, %s) = %v, want %v", tt.ac, tt.action, got, tt.want)
			}
		})
	}
}
