package http

import (
	"context"
	"net/http"
	"strings"

	"clinic-backend/internal/platform/jwt"
	"clinic-backend/internal/shared"
)

func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			RespondError(w, http.StatusUnauthorized, "unauthorized", "MISSING_TOKEN", nil)
			return
		}

		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
		claims, err := jwt.ValidateToken(tokenStr)
		if err != nil {
			RespondError(w, http.StatusUnauthorized, "invalid or expired token", "INVALID_TOKEN", err.Error())
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
