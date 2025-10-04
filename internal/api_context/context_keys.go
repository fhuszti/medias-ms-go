package api_context

import (
	"context"

	"github.com/fhuszti/medias-ms-go/internal/uuid"
)

type ctxKey string

const (
	IDKey         ctxKey = "id"
	AuthUserIDKey ctxKey = "authUserID"
	AuthRolesKey  ctxKey = "authRoles"
)

func IDFromContext(ctx context.Context) (uuid.UUID, bool) {
	id, ok := ctx.Value(IDKey).(uuid.UUID)
	return id, ok
}

func AuthUserIDFromContext(ctx context.Context) (uuid.UUID, bool) {
	id, ok := ctx.Value(AuthUserIDKey).(uuid.UUID)
	return id, ok
}

func AuthRolesFromContext(ctx context.Context) ([]string, bool) {
	roles, ok := ctx.Value(AuthRolesKey).([]string)
	return roles, ok
}
