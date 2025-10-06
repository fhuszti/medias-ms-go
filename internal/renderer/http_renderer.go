package renderer

import (
	"context"
	"encoding/json"
	"fmt"
	"hash/crc32"

	"github.com/fhuszti/medias-ms-go/internal/port"
	"github.com/fhuszti/medias-ms-go/internal/uuid"

	"github.com/fhuszti/medias-ms-go/internal/logger"
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
func (r *httpRenderer) RenderGetMedia(ctx context.Context, getter port.MediaGetter, id uuid.UUID) ([]byte, string, error) {
	raw, err := r.cache.GetMediaDetails(ctx, id)
	etag, errEtag := r.cache.GetEtagMediaDetails(ctx, id)
	if err == nil && errEtag == nil && raw != nil && etag != "" {
		logger.Infof(ctx, "http renderer used the cache to return details for media #%s", id)
		return raw, etag, nil
	}

	out, err := getter.GetMedia(ctx, id)
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
