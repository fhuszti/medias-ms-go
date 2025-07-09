package port

import (
	"context"
	"time"

	"github.com/fhuszti/medias-ms-go/internal/model"
	"github.com/fhuszti/medias-ms-go/internal/uuid"
)

// MediaRepository defines persistence operations for medias.
type MediaRepository interface {
	Create(ctx context.Context, media *model.Media) error
	Update(ctx context.Context, media *model.Media) error
	GetByID(ctx context.Context, ID uuid.UUID) (*model.Media, error)
	Delete(ctx context.Context, ID uuid.UUID) error
	ListUnoptimisedCompletedBefore(ctx context.Context, before time.Time) ([]uuid.UUID, error)
	ListOptimisedImagesNoVariantsBefore(ctx context.Context, before time.Time) ([]uuid.UUID, error)
}
