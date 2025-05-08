package media

import (
	"context"
	"time"
)

type Storage interface {
	GeneratePresignedDownloadURL(ctx context.Context, objectKey string, expiry time.Duration, downloadName string, inline bool) (string, error)
	GeneratePresignedUploadURL(ctx context.Context, objectKey string, expiry time.Duration) (string, error)
	ObjectExists(ctx context.Context, objectKey string) (bool, error)
	PublicURL(objectKey string) string
}
