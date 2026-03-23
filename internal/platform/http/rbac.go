package http

import (
	"net/http"

	"clinic-backend/internal/shared"
)

// RBACMiddleware enforces that the user has one of the allowed roles.
func RBACMiddleware(allowedRoles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userCtx, ok := shared.GetUserContext(r.Context())
			if !ok {
				RespondError(w, http.StatusUnauthorized, "unauthorized", "UNAUTHORIZED", nil)
				return
			}

			hasAccess := false
			for _, role := range allowedRoles {
				if userCtx.Role == role {
					hasAccess = true
					break
				}
			}

			if !hasAccess {
				RespondError(w, http.StatusForbidden, "forbidden: insufficient privileges", "FORBIDDEN", nil)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
