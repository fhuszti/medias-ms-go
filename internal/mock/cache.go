package mock

import (
	"context"
	"time"

	"github.com/fhuszti/medias-ms-go/internal/db"
)

// MockCache implements cache behaviour for tests.
type MockCache struct {
	Data []byte

	GetMediaErr error
	DelMediaErr error

	GetMediaCalled bool
	SetMediaCalled bool
	DelMediaCalled bool
	DelEtagCalled  bool
}

func (c *MockCache) GetMediaDetails(ctx context.Context, id db.UUID) ([]byte, error) {
	c.GetMediaCalled = true
	if c.GetMediaErr != nil {
		return nil, c.GetMediaErr
	}
	return c.Data, nil
}

func (c *MockCache) GetEtagMediaDetails(ctx context.Context, id db.UUID) (string, error) {
	return "", nil
}

func (c *MockCache) SetMediaDetails(ctx context.Context, id db.UUID, data []byte, validUntil time.Time) {
	c.SetMediaCalled = true
	c.Data = data
}

func (c *MockCache) SetEtagMediaDetails(ctx context.Context, id db.UUID, etag string, validUntil time.Time) {
}

func (c *MockCache) DeleteMediaDetails(ctx context.Context, id db.UUID) error {
	c.DelMediaCalled = true
	return c.DelMediaErr
}

func (c *MockCache) DeleteEtagMediaDetails(ctx context.Context, id db.UUID) error {
	c.DelEtagCalled = true
	return nil
}
