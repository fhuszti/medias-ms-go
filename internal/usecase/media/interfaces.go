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
	Delete(ctx context.Context, ID db.UUID) error
	ListUnoptimisedCompletedBefore(ctx context.Context, before time.Time) ([]db.UUID, error)
	ListOptimisedImagesNoVariantsBefore(ctx context.Context, before time.Time) ([]db.UUID, error)
}

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

type Cache interface {
	GetMediaDetails(ctx context.Context, id db.UUID) (*GetMediaOutput, error)
	GetEtagMediaDetails(ctx context.Context, id db.UUID) (string, error)
	SetMediaDetails(ctx context.Context, id db.UUID, value *GetMediaOutput)
	DeleteMediaDetails(ctx context.Context, id db.UUID) error
}

type FileOptimiser interface {
	Compress(mimeType string, r io.Reader) (io.ReadCloser, string, error)
	Resize(mimeType string, r io.Reader, width, height int) (io.ReadCloser, error)
}

type TaskDispatcher interface {
	EnqueueOptimiseMedia(ctx context.Context, id db.UUID) error
	EnqueueResizeImage(ctx context.Context, id db.UUID) error
}
