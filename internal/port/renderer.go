package port

import (
	"context"

	"github.com/fhuszti/medias-ms-go/internal/db"
)

// HTTPRenderer mediates between HTTP handlers and the media getter use case.
// It provides caching capabilities and returns both the JSON representation of
// the result as well as an ETag value derived from it.
type HTTPRenderer interface {
	// RenderGetMedia returns the cached JSON result and its ETag if available or
	// executes the underlying use case and caches the output otherwise.
	RenderGetMedia(ctx context.Context, getter MediaGetter, id db.UUID) ([]byte, string, error)
}
