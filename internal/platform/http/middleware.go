package http

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"clinic-backend/internal/platform/jwt"
	"clinic-backend/internal/shared"
)

// AuthMiddleware validates the Bearer token on every protected request.
//
// Error codes returned:
//   - MISSING_TOKEN   — no Authorization header or not a Bearer token.
//   - TOKEN_EXPIRED   — valid signature but past ExpiresAt; frontend should
//                       attempt a silent refresh or redirect to login.
//   - INVALID_TOKEN   — structurally bad, tampered, or wrong signing key.
//
// Only INVALID_TOKEN is a potential security event worth logging.
// TOKEN_EXPIRED is a normal lifecycle event and must NOT be logged as an error.
func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			RespondError(w, http.StatusUnauthorized, "missing or malformed authorization header", "MISSING_TOKEN", nil)
			return
		}

		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
		claims, err := jwt.ValidateToken(tokenStr)
		if err != nil {
			if errors.Is(err, jwt.ErrTokenExpired) {
				// Expected lifecycle event — no log noise, specific code for frontend.
				RespondError(w, http.StatusUnauthorized, "session expired, please refresh your token or log in again", "TOKEN_EXPIRED", nil)
				return
			}
			// Genuinely invalid token — could indicate tampering; worth logging.
			RespondError(w, http.StatusUnauthorized, "invalid token", "INVALID_TOKEN", nil)
			return
		}

		userCtx := &shared.UserContext{
			UserID:   claims.UserID,
			TenantID: claims.TenantID,
			Role:     claims.Role,
		}

		ctx := context.WithValue(r.Context(), shared.UserContextKey, userCtx)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
