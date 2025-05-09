package media

import (
	"context"
	"github.com/fhuszti/medias-ms-go/internal/model"
)

type Repository interface {
	Create(ctx context.Context, media *model.Media) error
}
