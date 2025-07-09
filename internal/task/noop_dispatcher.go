package task

import (
	"context"

	"github.com/fhuszti/medias-ms-go/internal/port"
	"github.com/fhuszti/medias-ms-go/internal/uuid"
)

type NoopDispatcher struct{}

var _ port.TaskDispatcher = (*NoopDispatcher)(nil)

func NewNoopDispatcher() *NoopDispatcher { return &NoopDispatcher{} }

func (d *NoopDispatcher) EnqueueOptimiseMedia(ctx context.Context, id uuid.UUID) error {
	return nil
}

func (d *NoopDispatcher) EnqueueResizeImage(ctx context.Context, id uuid.UUID) error {
	return nil
}
