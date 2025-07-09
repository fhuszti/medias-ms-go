package mock

import (
	"context"
	"time"

	"github.com/fhuszti/medias-ms-go/internal/uuid"
)

// MockCache implements cache behaviour for tests.
type MockCache struct {
	Data []byte
	Etag string

	GetMediaErr error
	GetEtagErr  error
	DelMediaErr error

	GetMediaCalled bool
	GetEtagCalled  bool
	SetMediaCalled bool
	SetEtagCalled  bool
	DelMediaCalled bool
	DelEtagCalled  bool
}

func (c *MockCache) GetMediaDetails(ctx context.Context, id uuid.UUID) ([]byte, error) {
	c.GetMediaCalled = true
	if c.GetMediaErr != nil {
		return nil, c.GetMediaErr
	}
	return c.Data, nil
}

func (c *MockCache) GetEtagMediaDetails(ctx context.Context, id uuid.UUID) (string, error) {
	c.GetEtagCalled = true
	if c.GetEtagErr != nil {
		return "", c.GetEtagErr
	}
	return c.Etag, nil
}

func (c *MockCache) SetMediaDetails(ctx context.Context, id uuid.UUID, data []byte, validUntil time.Time) {
	c.SetMediaCalled = true
	c.Data = data
}

func (c *MockCache) SetEtagMediaDetails(ctx context.Context, id uuid.UUID, etag string, validUntil time.Time) {
	c.SetEtagCalled = true
	c.Etag = etag
}

func (c *MockCache) DeleteMediaDetails(ctx context.Context, id uuid.UUID) error {
	c.DelMediaCalled = true
	return c.DelMediaErr
}

func (c *MockCache) DeleteEtagMediaDetails(ctx context.Context, id uuid.UUID) error {
	c.DelEtagCalled = true
	return nil
}
