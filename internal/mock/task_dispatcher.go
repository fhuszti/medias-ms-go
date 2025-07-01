package mock

import (
	"context"

	"github.com/fhuszti/medias-ms-go/internal/db"
)

// MockDispatcher implements task dispatching for tests.
type MockDispatcher struct {
	OptimiseCalled bool
	OptimiseIDs    []db.UUID
	OptimiseErr    error

	ResizeCalled bool
	ResizeIDs    []db.UUID
	ResizeErr    error
}

func (m *MockDispatcher) EnqueueOptimiseMedia(ctx context.Context, id db.UUID) error {
	m.OptimiseCalled = true
	m.OptimiseIDs = append(m.OptimiseIDs, id)
	return m.OptimiseErr
}

func (m *MockDispatcher) EnqueueResizeImage(ctx context.Context, id db.UUID) error {
	m.ResizeCalled = true
	m.ResizeIDs = append(m.ResizeIDs, id)
	return m.ResizeErr
}
