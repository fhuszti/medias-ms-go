package media

import (
	"context"
	"github.com/fhuszti/medias-ms-go/internal/db"
	"github.com/fhuszti/medias-ms-go/internal/model"
	"io"
	"time"
)

type FileInfo struct {
	SizeBytes   int64
	ContentType string
}

type Repository interface {
	Create(ctx context.Context, media *model.Media) error
	Update(ctx context.Context, media *model.Media) error
	GetByID(ctx context.Context, ID db.UUID) (*model.Media, error)
}

type Storage interface {
	GeneratePresignedDownloadURL(ctx context.Context, fileKey string, expiry time.Duration) (string, error)
	GeneratePresignedUploadURL(ctx context.Context, fileKey string, expiry time.Duration) (string, error)
	FileExists(ctx context.Context, fileKey string) (bool, error)
	StatFile(ctx context.Context, fileKey string) (FileInfo, error)
	RemoveFile(ctx context.Context, fileKey string) error
	GetFile(ctx context.Context, fileKey string) (io.ReadSeekCloser, error)
	SaveFile(ctx context.Context, fileKey string, reader io.Reader, fileSize int64, opts map[string]string) error
	CopyFile(ctx context.Context, srcKey, destKey string) error
}

type StorageGetter func(bucket string) (Storage, error)

type Cache interface {
	GetMediaDetails(ctx context.Context, id db.UUID) (*GetMediaOutput, error)
	SetMediaDetails(ctx context.Context, id db.UUID, value *GetMediaOutput)
}

type FileOptimiser interface {
	Compress(mimeType string, r io.Reader) (io.ReadCloser, string, error)
	Resize(mimeType string, r io.Reader, width, height int) ([]byte, error)
}
