package port

import (
	"context"

	"github.com/fhuszti/medias-ms-go/internal/db"
	media "github.com/fhuszti/medias-ms-go/internal/usecase/media"
)

// MediaGetter retrieves media information from the repository and storage.
type MediaGetter interface {
	GetMedia(ctx context.Context, id db.UUID) (*media.GetMediaOutput, error)
}
