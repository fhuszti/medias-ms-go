package port

import (
	"context"

	"github.com/fhuszti/medias-ms-go/internal/db"
)

// TaskDispatcher enqueues asynchronous tasks related to media processing.
type TaskDispatcher interface {
	EnqueueOptimiseMedia(ctx context.Context, id db.UUID) error
	EnqueueResizeImage(ctx context.Context, id db.UUID) error
}
