package cache

import (
	"context"
	"github.com/fhuszti/medias-ms-go/internal/db"
	"github.com/fhuszti/medias-ms-go/internal/usecase/media"
)

type NoopCache struct{}

// compile-time check: *NoopCache must satisfy media.Cache
var _ media.Cache = (*NoopCache)(nil)

func NewNoop() *NoopCache {
	return &NoopCache{}
}

func (n *NoopCache) GetMediaDetails(ctx context.Context, id db.UUID) (*media.GetMediaOutput, error) {
	return nil, nil // always cache miss
}

func (n *NoopCache) SetMediaDetails(ctx context.Context, id db.UUID, mOut *media.GetMediaOutput) {}

func (n *NoopCache) DeleteMediaDetails(ctx context.Context, id db.UUID) error { return nil }
