package api

import (
	"context"
	"github.com/fhuszti/medias-ms-go/internal/db"
)

type ctxKey string

const DestBucketKey ctxKey = "destBucket"
const IDKey ctxKey = "id"

func BucketFromContext(ctx context.Context) (string, bool) {
	b, ok := ctx.Value(DestBucketKey).(string)
	return b, ok
}

func IDFromContext(ctx context.Context) (db.UUID, bool) {
	id, ok := ctx.Value(IDKey).(db.UUID)
	return id, ok
}
