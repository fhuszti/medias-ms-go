package port

import (
	"context"
	"time"

	"github.com/fhuszti/medias-ms-go/internal/uuid"
)

// Cache provides caching capabilities for media retrieval.
type Cache interface {
	GetMediaDetails(ctx context.Context, id uuid.UUID) ([]byte, error)
	GetEtagMediaDetails(ctx context.Context, id uuid.UUID) (string, error)
	SetMediaDetails(ctx context.Context, id uuid.UUID, data []byte, validUntil time.Time)
	SetEtagMediaDetails(ctx context.Context, id uuid.UUID, etag string, validUntil time.Time)
	DeleteMediaDetails(ctx context.Context, id uuid.UUID) error
	DeleteEtagMediaDetails(ctx context.Context, id uuid.UUID) error
}
