package port

import (
	"context"

	"github.com/fhuszti/medias-ms-go/internal/uuid"
)

// TaskDispatcher enqueues asynchronous tasks related to media processing.
type TaskDispatcher interface {
	EnqueueOptimiseMedia(ctx context.Context, id uuid.UUID) error
	EnqueueResizeImage(ctx context.Context, id uuid.UUID) error
}
