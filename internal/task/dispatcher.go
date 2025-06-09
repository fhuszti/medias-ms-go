package task

import (
	"context"

	"github.com/fhuszti/medias-ms-go/internal/db"
	media "github.com/fhuszti/medias-ms-go/internal/usecase/media"
	"github.com/hibiken/asynq"
)

type Dispatcher struct {
	client *asynq.Client
}

// compile-time check
var _ media.TaskDispatcher = (*Dispatcher)(nil)

func NewDispatcher(addr, password string) *Dispatcher {
	c := asynq.NewClient(asynq.RedisClientOpt{Addr: addr, Password: password})
	return &Dispatcher{client: c}
}

func (d *Dispatcher) EnqueueOptimiseMedia(ctx context.Context, id db.UUID) error {
	t, err := NewOptimiseMediaTask(id.String())
	if err != nil {
		return err
	}
	if _, err := d.client.EnqueueContext(ctx, t); err != nil {
		return err
	}
	return nil
}
