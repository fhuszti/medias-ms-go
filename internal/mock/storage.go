package mock

import (
	"bytes"
	"context"
	"io"
	"time"

	"github.com/fhuszti/medias-ms-go/internal/port"
)

// MockStorage implements the storage interface for tests.
type MockStorage struct {
	Reader    io.ReadSeeker
	StatInfo  port.FileInfo
	ObjectKey string
	TTL       time.Duration
	Exists    bool

	InitBucketErr           error
	GenerateDownloadLinkErr error
	GenerateUploadLinkErr   error
	StatErr                 error
	GetErr                  error
	SaveErr                 error
	CopyErr                 error
	RemoveErr               error
	FileExistsErr           error

	InitBucketCalled           bool
	GenerateDownloadLinkCalled bool
	GenerateUploadLinkCalled   bool
	StatCalled                 bool
	GetCalled                  bool
	SaveCalled                 bool
	RemoveCalled               bool
	CopyCalled                 bool
	FileExistsCalled           bool
}

func (m *MockStorage) InitBucket(bucket string) error {
	m.InitBucketCalled = true
	return m.InitBucketErr
}

func (m *MockStorage) GeneratePresignedDownloadURL(ctx context.Context, bucket, fileKey string, expiry time.Duration) (string, error) {
	m.GenerateDownloadLinkCalled = true
	m.ObjectKey = fileKey
	m.TTL = expiry
	if m.GenerateDownloadLinkErr != nil {
		return "", m.GenerateDownloadLinkErr
	}
	return "https://example.com/upload", nil
}

func (m *MockStorage) GeneratePresignedUploadURL(ctx context.Context, bucket, fileKey string, expiry time.Duration) (string, error) {
	m.GenerateUploadLinkCalled = true
	m.ObjectKey = fileKey
	m.TTL = expiry
	if m.GenerateUploadLinkErr != nil {
		return "", m.GenerateUploadLinkErr
	}
	return "https://example.com/upload", nil
}

func (m *MockStorage) StatFile(ctx context.Context, bucket, fileKey string) (port.FileInfo, error) {
	m.StatCalled = true
	if m.StatErr != nil {
		return port.FileInfo{}, m.StatErr
	}
	return m.StatInfo, nil
}

func (m *MockStorage) RemoveFile(ctx context.Context, bucket, fileKey string) error {
	m.RemoveCalled = true
	return m.RemoveErr
}

func (m *MockStorage) GetFile(ctx context.Context, bucket, fileKey string) (io.ReadSeekCloser, error) {
	m.GetCalled = true
	if m.GetErr != nil {
		return nil, m.GetErr
	}
	if m.Reader != nil {
		return noopRSC{m.Reader}, nil
	}
	return noopRSC{bytes.NewReader([]byte("dummy"))}, nil
}

func (m *MockStorage) SaveFile(ctx context.Context, bucket, fileKey string, reader io.Reader, fileSize int64, opts map[string]string) error {
	m.SaveCalled = true
	return m.SaveErr
}

func (m *MockStorage) CopyFile(ctx context.Context, bucket, srcKey, destKey string) error {
	m.CopyCalled = true
	return m.CopyErr
}

func (m *MockStorage) FileExists(ctx context.Context, bucket, fileKey string) (bool, error) {
	m.FileExistsCalled = true
	if m.FileExistsErr != nil {
		return false, m.FileExistsErr
	}
	return m.Exists, nil
}
