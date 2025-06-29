package port

import (
	"context"

	"github.com/fhuszti/medias-ms-go/internal/db"
)

// Cache provides caching capabilities for media retrieval.
type Cache interface {
	GetMediaDetails(ctx context.Context, id db.UUID) (*GetMediaOutput, error)
	GetEtagMediaDetails(ctx context.Context, id db.UUID) (string, error)
	SetMediaDetails(ctx context.Context, id db.UUID, value *GetMediaOutput)
	DeleteMediaDetails(ctx context.Context, id db.UUID) error
}
