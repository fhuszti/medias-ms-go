package api

import (
	"context"
	"github.com/fhuszti/medias-ms-go/internal/db"
)

type ctxKey string

const IDKey ctxKey = "id"

func IDFromContext(ctx context.Context) (db.UUID, bool) {
	id, ok := ctx.Value(IDKey).(db.UUID)
	return id, ok
}
