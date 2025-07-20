package mock

import (
	"context"

	"github.com/fhuszti/medias-ms-go/internal/uuid"
)

// Dispatcher implements task dispatching for tests.
type Dispatcher struct {
	// captured inputs
	OptimiseIDs []uuid.UUID
	ResizeIDs   []uuid.UUID

	// errors
	OptimiseErr error
	ResizeErr   error

	// call flags
	OptimiseCalled bool
	ResizeCalled   bool
}

func (m *Dispatcher) EnqueueOptimiseMedia(ctx context.Context, id uuid.UUID) error {
	m.OptimiseCalled = true
	m.OptimiseIDs = append(m.OptimiseIDs, id)
	return m.OptimiseErr
}

func (m *Dispatcher) EnqueueResizeImage(ctx context.Context, id uuid.UUID) error {
	m.ResizeCalled = true
	m.ResizeIDs = append(m.ResizeIDs, id)
	return m.ResizeErr
}
