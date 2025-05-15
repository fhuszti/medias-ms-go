package media

import "context"

type ctxKey string

const DestBucketKey ctxKey = "destBucket"

func BucketFromContext(ctx context.Context) (string, bool) {
	b, ok := ctx.Value(DestBucketKey).(string)
	return b, ok
}
