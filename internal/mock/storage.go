package mock

import (
	"bytes"
	"context"
	"io"
	"time"

	"github.com/fhuszti/medias-ms-go/internal/port"
)

// Storage implements the storage interface for tests.
type Storage struct {
	// stored values
	StatInfoOut port.FileInfo
	GetOut      io.ReadSeeker
	ExistsOut   bool

	// captured inputs
	ObjectKey string
	TTL       time.Duration

	// errors
	InitBucketErr           error
	GenerateDownloadLinkErr error
	GenerateUploadLinkErr   error
	StatErr                 error
	RemoveErr               error
	GetErr                  error
	SaveErr                 error
	CopyErr                 error
	FileExistsErr           error

	// call flags
	InitBucketCalled           bool
	GenerateDownloadLinkCalled bool
	GenerateUploadLinkCalled   bool
	StatCalled                 bool
	RemoveCalled               bool
	GetCalled                  bool
	SaveCalled                 bool
	CopyCalled                 bool
	FileExistsCalled           bool
}

func (m *Storage) InitBucket(bucket string) error {
	m.InitBucketCalled = true
	return m.InitBucketErr
}

func (m *Storage) GeneratePresignedDownloadURL(ctx context.Context, bucket, fileKey string, expiry time.Duration) (string, error) {
	m.GenerateDownloadLinkCalled = true
	m.ObjectKey = fileKey
	m.TTL = expiry
	if m.GenerateDownloadLinkErr != nil {
		return "", m.GenerateDownloadLinkErr
	}
	return "https://example.com/download", nil
}

func (m *Storage) GeneratePresignedUploadURL(ctx context.Context, bucket, fileKey string, expiry time.Duration) (string, error) {
	m.GenerateUploadLinkCalled = true
	m.ObjectKey = fileKey
	m.TTL = expiry
	if m.GenerateUploadLinkErr != nil {
		return "", m.GenerateUploadLinkErr
	}
	return "https://example.com/upload", nil
}

func (m *Storage) StatFile(ctx context.Context, bucket, fileKey string) (port.FileInfo, error) {
	m.StatCalled = true
	if m.StatErr != nil {
		return port.FileInfo{}, m.StatErr
	}
	return m.StatInfoOut, nil
}

func (m *Storage) RemoveFile(ctx context.Context, bucket, fileKey string) error {
	m.RemoveCalled = true
	return m.RemoveErr
}

func (m *Storage) GetFile(ctx context.Context, bucket, fileKey string) (io.ReadSeekCloser, error) {
	m.GetCalled = true
	if m.GetErr != nil {
		return nil, m.GetErr
	}
	if m.GetOut != nil {
		return noopRSC{m.GetOut}, nil
	}
	return noopRSC{bytes.NewReader([]byte("dummy"))}, nil
}

func (m *Storage) SaveFile(ctx context.Context, bucket, fileKey string, reader io.Reader, fileSize int64, opts map[string]string) error {
	m.SaveCalled = true
	return m.SaveErr
}

func (m *Storage) CopyFile(ctx context.Context, bucket, srcKey, destKey string) error {
	m.CopyCalled = true
	return m.CopyErr
}

func (m *Storage) FileExists(ctx context.Context, bucket, fileKey string) (bool, error) {
	m.FileExistsCalled = true
	if m.FileExistsErr != nil {
		return false, m.FileExistsErr
	}
	return m.ExistsOut, nil
}
