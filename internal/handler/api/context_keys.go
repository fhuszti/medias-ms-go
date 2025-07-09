package api

import (
	"context"

	"github.com/fhuszti/medias-ms-go/internal/uuid"
)

type ctxKey string

const IDKey ctxKey = "id"

func IDFromContext(ctx context.Context) (uuid.UUID, bool) {
	id, ok := ctx.Value(IDKey).(uuid.UUID)
	return id, ok
}
