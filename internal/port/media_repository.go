package port

import (
	"context"
	"time"

	"github.com/fhuszti/medias-ms-go/internal/db"
	"github.com/fhuszti/medias-ms-go/internal/model"
)

// MediaRepository defines persistence operations for medias.
type MediaRepository interface {
	Create(ctx context.Context, media *model.Media) error
	Update(ctx context.Context, media *model.Media) error
	GetByID(ctx context.Context, ID db.UUID) (*model.Media, error)
	Delete(ctx context.Context, ID db.UUID) error
	ListUnoptimisedCompletedBefore(ctx context.Context, before time.Time) ([]db.UUID, error)
	ListOptimisedImagesNoVariantsBefore(ctx context.Context, before time.Time) ([]db.UUID, error)
}
