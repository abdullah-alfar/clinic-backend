package shared

import (
	"context"

	"github.com/google/uuid"
)

type ContextKey string

const (
	UserContextKey ContextKey = "userContext"
)

type UserContext struct {
	UserID   uuid.UUID
	TenantID uuid.UUID
	Role     string
}

// GetUserContext extracts UserContext from an http.Request context.
func GetUserContext(ctx context.Context) (*UserContext, bool) {
	uctx, ok := ctx.Value(UserContextKey).(*UserContext)
	return uctx, ok
}
