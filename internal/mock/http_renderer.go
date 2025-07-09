package mock

import (
	"context"

	"github.com/fhuszti/medias-ms-go/internal/port"
	"github.com/fhuszti/medias-ms-go/internal/uuid"
)

// MockHTTPRenderer implements port.HTTPRenderer for tests.
type MockHTTPRenderer struct {
	Data []byte
	Etag string
	Err  error

	Called bool
	Getter port.MediaGetter
	ID     uuid.UUID
}

func (m *MockHTTPRenderer) RenderGetMedia(ctx context.Context, getter port.MediaGetter, id uuid.UUID) ([]byte, string, error) {
	m.Called = true
	m.Getter = getter
	m.ID = id
	return m.Data, m.Etag, m.Err
}
