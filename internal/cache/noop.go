package cache

import (
	"context"
	"time"

	"github.com/fhuszti/medias-ms-go/internal/port"
	"github.com/fhuszti/medias-ms-go/internal/uuid"
)

type NoopCache struct{}

// compile-time check: *NoopCache must satisfy port.Cache
var _ port.Cache = (*NoopCache)(nil)

func NewNoop() *NoopCache {
	return &NoopCache{}
}

func (n *NoopCache) GetMediaDetails(ctx context.Context, id uuid.UUID) ([]byte, error) {
	return nil, nil // always cache miss
}

func (n *NoopCache) GetEtagMediaDetails(ctx context.Context, id uuid.UUID) (string, error) {
	return "", nil
}

func (n *NoopCache) SetMediaDetails(ctx context.Context, id uuid.UUID, data []byte, validUntil time.Time) {
}

func (n *NoopCache) SetEtagMediaDetails(ctx context.Context, id uuid.UUID, etag string, validUntil time.Time) {
}

func (n *NoopCache) DeleteMediaDetails(ctx context.Context, id uuid.UUID) error { return nil }

func (n *NoopCache) DeleteEtagMediaDetails(ctx context.Context, id uuid.UUID) error {
	return nil
}
