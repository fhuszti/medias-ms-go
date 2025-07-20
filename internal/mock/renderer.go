package mock

import (
	"context"

	"github.com/fhuszti/medias-ms-go/internal/port"
	"github.com/fhuszti/medias-ms-go/internal/uuid"
)

// HTTPRenderer implements port.HTTPRenderer for tests.
type HTTPRenderer struct {
	// stored values
	MediaOut []byte

	// etag values
	EtagMedia string

	// captured inputs
	GotMediaID uuid.UUID

	// errors
	GetMediaErr error

	// call flags
	GetMediaCalled bool
}

func (m *HTTPRenderer) RenderGetMedia(ctx context.Context, getter port.MediaGetter, id uuid.UUID) ([]byte, string, error) {
	m.GetMediaCalled = true
	m.GotMediaID = id
	return m.MediaOut, m.EtagMedia, m.GetMediaErr
}
