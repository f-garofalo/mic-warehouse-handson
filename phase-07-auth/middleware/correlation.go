package middleware

import (
	"context"

	"github.com/labstack/echo/v4"
)

// Correlation carries the cross-BC correlation keys per ADR-017.
type Correlation struct {
	TSID        string
	WorkspaceID string
}

type correlationKey struct{}

// HeaderTSID is the canonical header name for the TeamSystem user identity.
const HeaderTSID = "X-TS-ID"

// HeaderWorkspaceID is the canonical header name for the workspace correlation.
const HeaderWorkspaceID = "X-Workspace-ID"

// CorrelationMiddleware reads X-TS-ID and X-Workspace-ID from the request,
// falling back to the AuthContext if either is missing, attaches the result
// to the request context, and mirrors the headers onto the response.
func CorrelationMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		req := c.Request()
		corr := Correlation{
			TSID:        req.Header.Get(HeaderTSID),
			WorkspaceID: req.Header.Get(HeaderWorkspaceID),
		}
		if corr.TSID == "" || corr.WorkspaceID == "" {
			if ac := AuthContextFrom(req.Context()); ac != nil {
				if corr.TSID == "" {
					corr.TSID = ac.TSID
				}
				if corr.WorkspaceID == "" {
					corr.WorkspaceID = ac.WorkspaceID
				}
			}
		}
		ctx := context.WithValue(req.Context(), correlationKey{}, corr)
		c.SetRequest(req.WithContext(ctx))

		if corr.TSID != "" {
			c.Response().Header().Set(HeaderTSID, corr.TSID)
		}
		if corr.WorkspaceID != "" {
			c.Response().Header().Set(HeaderWorkspaceID, corr.WorkspaceID)
		}
		return next(c)
	}
}

// CorrelationFrom returns the Correlation attached to a context, or the zero value if absent.
func CorrelationFrom(ctx context.Context) Correlation {
	if v := ctx.Value(correlationKey{}); v != nil {
		if c, ok := v.(Correlation); ok {
			return c
		}
	}
	return Correlation{}
}
