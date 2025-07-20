package mock

import (
	"context"
	"time"

	"github.com/fhuszti/medias-ms-go/internal/uuid"
)

// Cache implements cache behaviour for tests.
type Cache struct {
	// stored values
	MediaOut []byte

	// etag values
	EtagMedia string

	// errors
	GetMediaErr     error
	GetEtagMediaErr error
	DelMediaErr     error
	DelEtagMediaErr error

	// call flags
	GetMediaCalled     bool
	GetEtagMediaCalled bool
	SetMediaCalled     bool
	SetEtagMediaCalled bool
	DelMediaCalled     bool
	DelEtagMediaCalled bool
}

func (c *Cache) GetMediaDetails(ctx context.Context, id uuid.UUID) ([]byte, error) {
	c.GetMediaCalled = true
	if c.GetMediaErr != nil {
		return nil, c.GetMediaErr
	}
	return c.MediaOut, nil
}

func (c *Cache) GetEtagMediaDetails(ctx context.Context, id uuid.UUID) (string, error) {
	c.GetEtagMediaCalled = true
	if c.GetEtagMediaErr != nil {
		return "", c.GetEtagMediaErr
	}
	return c.EtagMedia, nil
}

func (c *Cache) SetMediaDetails(ctx context.Context, id uuid.UUID, data []byte, validUntil time.Time) {
	c.SetMediaCalled = true
	c.MediaOut = data
}

func (c *Cache) SetEtagMediaDetails(ctx context.Context, id uuid.UUID, etag string, validUntil time.Time) {
	c.SetEtagMediaCalled = true
	c.EtagMedia = etag
}

func (c *Cache) DeleteMediaDetails(ctx context.Context, id uuid.UUID) error {
	c.DelMediaCalled = true
	return c.DelMediaErr
}

func (c *Cache) DeleteEtagMediaDetails(ctx context.Context, id uuid.UUID) error {
	c.DelEtagMediaCalled = true
	return c.DelEtagMediaErr
}
