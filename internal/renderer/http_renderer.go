package renderer

import (
	"context"
	"encoding/json"
	"fmt"
	"hash/crc32"

	"github.com/fhuszti/medias-ms-go/internal/db"
	"github.com/fhuszti/medias-ms-go/internal/port"
)

type httpRenderer struct {
	cache port.Cache
}

// compile-time check: *httpRenderer must satisfy port.HTTPRenderer
var _ port.HTTPRenderer = (*httpRenderer)(nil)

// NewHTTPRenderer creates a new HTTPRenderer implementation.
func NewHTTPRenderer(cache port.Cache) port.HTTPRenderer {
	return &httpRenderer{cache: cache}
}

// RenderGetMedia fetches media details either from cache or from the wrapped use
// case. It returns the JSON encoded output and a quoted ETag string.
func (r *httpRenderer) RenderGetMedia(ctx context.Context, getter port.MediaGetter, id db.UUID) ([]byte, string, error) {
	raw, err := r.cache.GetMediaDetails(ctx, id)
	etag, errEtag := r.cache.GetEtagMediaDetails(ctx, id)
	if err == nil && errEtag == nil && raw != nil && etag != "" {
		return raw, etag, nil
	}

	out, err := getter.GetMedia(ctx, port.GetMediaInput{ID: id})
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
