package port

import (
	"context"
	"io"
	"time"
)

// FileInfo represents metadata about a stored file.
type FileInfo struct {
	SizeBytes   int64
	ContentType string
}

// Storage defines file storage operations.
type Storage interface {
	InitBucket(bucket string) error
	GeneratePresignedDownloadURL(ctx context.Context, bucket, fileKey string, expiry time.Duration) (string, error)
	GeneratePresignedUploadURL(ctx context.Context, bucket, fileKey string, expiry time.Duration) (string, error)
	FileExists(ctx context.Context, bucket, fileKey string) (bool, error)
	StatFile(ctx context.Context, bucket, fileKey string) (FileInfo, error)
	RemoveFile(ctx context.Context, bucket, fileKey string) error
	GetFile(ctx context.Context, bucket, fileKey string) (io.ReadSeekCloser, error)
	SaveFile(ctx context.Context, bucket, fileKey string, reader io.Reader, fileSize int64, opts map[string]string) error
	CopyFile(ctx context.Context, bucket, srcKey, destKey string) error
}
