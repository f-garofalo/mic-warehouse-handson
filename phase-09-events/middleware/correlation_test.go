package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
)

func TestCorrelationMiddleware_readsHeadersAndAttachesToContext(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-TS-ID", "ts-42")
	req.Header.Set("X-Workspace-ID", "ws-7")
	rec := httptest.NewRecorder()
	e := echo.New()
	c := e.NewContext(req, rec)

	called := false
	next := func(c echo.Context) error {
		called = true
		ac := CorrelationFrom(c.Request().Context())
		if ac.TSID != "ts-42" || ac.WorkspaceID != "ws-7" {
			t.Errorf("unexpected correlation: %+v", ac)
		}
		return nil
	}
	if err := CorrelationMiddleware(next)(c); err != nil {
		t.Fatalf("CorrelationMiddleware: %v", err)
	}
	if !called {
		t.Fatal("expected next handler to be called")
	}
	if got := rec.Header().Get("X-TS-ID"); got != "ts-42" {
		t.Errorf("expected response X-TS-ID=ts-42, got %q", got)
	}
	if got := rec.Header().Get("X-Workspace-ID"); got != "ws-7" {
		t.Errorf("expected response X-Workspace-ID=ws-7, got %q", got)
	}
}

func TestCorrelationMiddleware_picksUpFromAuthContextWhenHeadersMissing(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := echo.New().NewContext(req, rec)

	ac := &AuthContext{Kind: AuthKindUser, UserID: "u-1", TSID: "ts-from-auth", WorkspaceID: "ws-from-auth"}
	c.SetRequest(req.WithContext(WithAuthContext(req.Context(), ac)))

	next := func(c echo.Context) error {
		corr := CorrelationFrom(c.Request().Context())
		if corr.TSID != "ts-from-auth" || corr.WorkspaceID != "ws-from-auth" {
			t.Errorf("expected correlation from auth context, got %+v", corr)
		}
		return nil
	}
	if err := CorrelationMiddleware(next)(c); err != nil {
		t.Fatalf("CorrelationMiddleware: %v", err)
	}
}

func TestCorrelationFrom_emptyContextReturnsEmpty(t *testing.T) {
	corr := CorrelationFrom(context.Background())
	if corr.TSID != "" || corr.WorkspaceID != "" {
		t.Errorf("expected empty correlation, got %+v", corr)
	}
}
