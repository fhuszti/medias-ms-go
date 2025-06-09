package task

import (
	"context"

	"github.com/fhuszti/medias-ms-go/internal/db"
	media "github.com/fhuszti/medias-ms-go/internal/usecase/media"
)

type NoopDispatcher struct{}

var _ media.TaskDispatcher = (*NoopDispatcher)(nil)

func NewNoopDispatcher() *NoopDispatcher { return &NoopDispatcher{} }

func (d *NoopDispatcher) EnqueueOptimiseMedia(ctx context.Context, id db.UUID) error {
	return nil
}
