package renderer

import (
	"context"
	"encoding/json"
	"fmt"
	"hash/crc32"

	"github.com/fhuszti/medias-ms-go/internal/db"
	"github.com/fhuszti/medias-ms-go/internal/port"
	"github.com/fhuszti/medias-ms-go/internal/usecase/media"
)

// HTTPRenderer mediates between HTTP handlers and the media getter use case.
// It provides caching capabilities and returns both the JSON representation of
// the result as well as an ETag value derived from it.
type HTTPRenderer interface {
	// RenderGetMedia returns the cached JSON result and its ETag if available or
	// executes the underlying use case and caches the output otherwise.
	RenderGetMedia(ctx context.Context, getter media.Getter, id db.UUID) ([]byte, string, error)
}

type httpRenderer struct {
	cache port.Cache
}

// compile-time check: *httpRenderer must satisfy HTTPRenderer
var _ HTTPRenderer = (*httpRenderer)(nil)

// NewHTTPRenderer creates a new HTTPRenderer implementation.
func NewHTTPRenderer(cache port.Cache) HTTPRenderer {
	return &httpRenderer{cache: cache}
}

// RenderGetMedia fetches media details either from cache or from the wrapped use
// case. It returns the JSON encoded output and a quoted ETag string.
func (r *httpRenderer) RenderGetMedia(ctx context.Context, getter media.Getter, id db.UUID) ([]byte, string, error) {
	raw, err := r.cache.GetMediaDetails(ctx, id)
	etag, errEtag := r.cache.GetEtagMediaDetails(ctx, id)
	if err == nil && errEtag == nil && raw != nil && etag != "" {
		return raw, etag, nil
	}

	out, err := getter.GetMedia(ctx, media.GetMediaInput{ID: id})
	if err != nil {
		return nil, "", err
	}

	raw, err = json.Marshal(out)
	if err != nil {
		return nil, "", fmt.Errorf("json marshal: %w", err)
	}

	etag = fmt.Sprintf("\"%08x\"", crc32.ChecksumIEEE(raw))
	r.cache.SetMediaDetails(ctx, id, raw, out.ValidUntil)
	r.cache.SetEtagMediaDetails(ctx, id, etag, out.ValidUntil)

	return raw, etag, nil
}
