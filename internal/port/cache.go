package port

import (
	"context"
	"time"

	"github.com/fhuszti/medias-ms-go/internal/db"
)

// Cache provides caching capabilities for media retrieval.
type Cache interface {
	GetMediaDetails(ctx context.Context, id db.UUID) ([]byte, error)
	GetEtagMediaDetails(ctx context.Context, id db.UUID) (string, error)
	SetMediaDetails(ctx context.Context, id db.UUID, data []byte, validUntil time.Time)
	SetEtagMediaDetails(ctx context.Context, id db.UUID, etag string, validUntil time.Time)
	DeleteMediaDetails(ctx context.Context, id db.UUID) error
}
