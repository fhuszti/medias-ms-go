package mock

import (
	"context"

	"github.com/fhuszti/medias-ms-go/internal/db"
	"github.com/fhuszti/medias-ms-go/internal/port"
)

// MockMediaGetter implements port.MediaGetter for tests.
type MockMediaGetter struct {
	Out    *port.GetMediaOutput
	Err    error
	Called bool
}

func (m *MockMediaGetter) GetMedia(ctx context.Context, id db.UUID) (*port.GetMediaOutput, error) {
	m.Called = true
	return m.Out, m.Err
}
