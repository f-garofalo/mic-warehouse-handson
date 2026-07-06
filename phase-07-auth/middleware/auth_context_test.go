package middleware

import (
	"context"
	"testing"
)

func TestAuthContext_userKind(t *testing.T) {
	ac := AuthContext{
		Kind:        AuthKindUser,
		UserID:      "u-1",
		Email:       "alice@example.com",
		Scopes:      []string{"profile"},
		TSID:        "ts-42",
		WorkspaceID: "ws-7",
	}
	if !ac.IsUser() {
		t.Error("expected IsUser to be true")
	}
	if ac.IsM2M() {
		t.Error("expected IsM2M to be false")
	}
	if ac.Subject() != "u-1" {
		t.Errorf("expected Subject=u-1, got %s", ac.Subject())
	}
}

func TestAuthContext_m2mKind(t *testing.T) {
	ac := AuthContext{
		Kind:      AuthKindM2M,
		ServiceID: "svc-warehouse",
		Scopes:    []string{"platform.m2m"},
	}
	if !ac.IsM2M() {
		t.Error("expected IsM2M to be true")
	}
	if ac.IsUser() {
		t.Error("expected IsUser to be false")
	}
	if ac.Subject() != "svc-warehouse" {
		t.Errorf("expected Subject=svc-warehouse, got %s", ac.Subject())
	}
}

func TestAuthContext_putAndGet(t *testing.T) {
	ac := &AuthContext{Kind: AuthKindUser, UserID: "u-1"}
	ctx := WithAuthContext(context.Background(), ac)

	got := AuthContextFrom(ctx)
	if got == nil {
		t.Fatal("expected AuthContext, got nil")
	}
	if got.UserID != "u-1" {
		t.Errorf("expected UserID=u-1, got %s", got.UserID)
	}

	if AuthContextFrom(context.Background()) != nil {
		t.Error("expected nil AuthContext on empty context")
	}
}

func TestAuthContext_hasScope(t *testing.T) {
	ac := AuthContext{Scopes: []string{"profile", "platform.m2m"}}
	if !ac.HasScope("profile") {
		t.Error("expected HasScope(profile) = true")
	}
	if ac.HasScope("platform.admin") {
		t.Error("expected HasScope(platform.admin) = false")
	}
}
