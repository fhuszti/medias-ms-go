package task

import (
	"context"

	"github.com/fhuszti/medias-ms-go/internal/port"
	"github.com/fhuszti/medias-ms-go/internal/uuid"
	"github.com/hibiken/asynq"
)

type Dispatcher struct {
	client *asynq.Client
}

// compile-time check
var _ port.TaskDispatcher = (*Dispatcher)(nil)

func NewDispatcher(addr, password string) *Dispatcher {
	c := asynq.NewClient(asynq.RedisClientOpt{Addr: addr, Password: password})
	return &Dispatcher{client: c}
}

func (d *Dispatcher) EnqueueOptimiseMedia(ctx context.Context, id uuid.UUID) error {
	t, err := NewOptimiseMediaTask(id.String())
	if err != nil {
		return err
	}
	if _, err := d.client.EnqueueContext(ctx, t); err != nil {
		return err
	}
	return nil
}

func (d *Dispatcher) EnqueueResizeImage(ctx context.Context, id uuid.UUID) error {
	t, err := NewResizeImageTask(id.String())
	if err != nil {
		return err
	}
	if _, err := d.client.EnqueueContext(ctx, t); err != nil {
		return err
	}
	return nil
}
