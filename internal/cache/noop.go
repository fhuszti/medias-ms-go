package cache

import (
	"context"

	"github.com/fhuszti/medias-ms-go/internal/db"
	"github.com/fhuszti/medias-ms-go/internal/port"
)

type NoopCache struct{}

// compile-time check: *NoopCache must satisfy port.Cache
var _ port.Cache = (*NoopCache)(nil)

func NewNoop() *NoopCache {
	return &NoopCache{}
}

func (n *NoopCache) GetMediaDetails(ctx context.Context, id db.UUID) (*port.GetMediaOutput, error) {
	return nil, nil // always cache miss
}

func (n *NoopCache) GetEtagMediaDetails(ctx context.Context, id db.UUID) (string, error) {
	return "", nil
}

func (n *NoopCache) SetMediaDetails(ctx context.Context, id db.UUID, mOut *port.GetMediaOutput) {}

func (n *NoopCache) DeleteMediaDetails(ctx context.Context, id db.UUID) error { return nil }
