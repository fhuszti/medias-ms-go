package task

import (
	"context"

	"github.com/fhuszti/medias-ms-go/internal/db"
	"github.com/fhuszti/medias-ms-go/internal/port"
)

type NoopDispatcher struct{}

var _ port.TaskDispatcher = (*NoopDispatcher)(nil)

func NewNoopDispatcher() *NoopDispatcher { return &NoopDispatcher{} }

func (d *NoopDispatcher) EnqueueOptimiseMedia(ctx context.Context, id db.UUID) error {
	return nil
}

func (d *NoopDispatcher) EnqueueResizeImage(ctx context.Context, id db.UUID) error {
	return nil
}
