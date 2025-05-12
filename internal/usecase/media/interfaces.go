package media

import (
	"context"
	"github.com/fhuszti/medias-ms-go/internal/db"
	"github.com/fhuszti/medias-ms-go/internal/model"
	"github.com/minio/minio-go/v7"
	"io"
	"time"
)

type Repository interface {
	Create(ctx context.Context, media *model.Media) error
	Update(ctx context.Context, media *model.Media) error
	GetByID(ctx context.Context, ID db.UUID) (*model.Media, error)
}

type Storage interface {
	GeneratePresignedUploadURL(ctx context.Context, fileKey string, expiry time.Duration) (string, error)
	FileExists(ctx context.Context, fileKey string) (bool, error)
	StatFile(ctx context.Context, fileKey string) (minio.ObjectInfo, error)
	RemoveFile(ctx context.Context, fileKey string) error
	GetFile(ctx context.Context, fileKey string) (*minio.Object, error)
	SaveFile(ctx context.Context, fileKey string, reader io.Reader, fileSize int64, opts map[string]string) (minio.UploadInfo, error)
}

type StorageGetter func(bucket string) (Storage, error)
