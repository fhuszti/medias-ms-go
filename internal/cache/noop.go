package cache

import (
	"context"
	"time"

	"github.com/fhuszti/medias-ms-go/internal/db"
	"github.com/fhuszti/medias-ms-go/internal/port"
)

type NoopCache struct{}

// compile-time check: *NoopCache must satisfy port.Cache
var _ port.Cache = (*NoopCache)(nil)

func NewNoop() *NoopCache {
	return &NoopCache{}
}

func (n *NoopCache) GetMediaDetails(ctx context.Context, id db.UUID) ([]byte, error) {
	return nil, nil // always cache miss
}

func (n *NoopCache) GetEtagMediaDetails(ctx context.Context, id db.UUID) (string, error) {
	return "", nil
}

func (n *NoopCache) SetMediaDetails(ctx context.Context, id db.UUID, data []byte, validUntil time.Time) {
}

func (n *NoopCache) SetEtagMediaDetails(ctx context.Context, id db.UUID, etag string, validUntil time.Time) {
}

func (n *NoopCache) DeleteMediaDetails(ctx context.Context, id db.UUID) error { return nil }

func (n *NoopCache) DeleteEtagMediaDetails(ctx context.Context, id db.UUID) error {
	return nil
}
