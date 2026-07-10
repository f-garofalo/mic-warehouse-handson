package middleware

import "testing"

func TestTSIDClaims_userContext(t *testing.T) {
	claims := tsidClaims{
		Subject:          "u-1",
		Email:            "alice@example.com",
		NCSID:            "ncs-1",
		ClientID:         "tsid-client-demo",
		IdentityProvider: "teamsystem",
		Scopes:           []string{"openid", "profile"},
		AMR:              []string{"external"},
	}
	ac := claims.AuthContext()
	if ac.Kind != AuthKindUser || ac.UserID != "u-1" {
		t.Fatalf("expected user context, got %+v", ac)
	}
	// TSID is derived from the OAuth `sub` claim, not `ncs_id` (which is a
	// Notification Center Service routing id, not an identity).
	if ac.TSID != "u-1" {
		t.Errorf("expected TSID from sub, got %s", ac.TSID)
	}
	if !ac.HasScope("profile") {
		t.Errorf("expected profile scope, got %v", ac.Scopes)
	}
}

func TestTSIDClaims_m2mContext(t *testing.T) {
	claims := tsidClaims{
		Subject:  "svc-orders",
		ClientID: "svc-orders",
		Scopes:   []string{"platform.m2m"},
		AMR:      []string{"client_credentials"},
	}
	ac := claims.AuthContext()
	if ac.Kind != AuthKindM2M || ac.ServiceID != "svc-orders" {
		t.Fatalf("expected M2M context, got %+v", ac)
	}
	if ac.Subject() != "svc-orders" {
		t.Errorf("expected subject svc-orders, got %s", ac.Subject())
	}
}
