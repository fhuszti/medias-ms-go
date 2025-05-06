package service

import (
	"context"
	"fmt"
	"github.com/fhuszti/medias-ms-go/internal/db"
	"github.com/fhuszti/medias-ms-go/internal/model"
	"time"
)

type MediaRepository interface {
	Create(ctx context.Context, media *model.Media) error
}

type Storage interface {
	GeneratePresignedDownloadURL(ctx context.Context, objectKey string, expiry time.Duration, downloadName string, inline bool) (string, error)
	GeneratePresignedUploadURL(ctx context.Context, objectKey string, expiry time.Duration) (string, error)
	ObjectExists(ctx context.Context, objectKey string) (bool, error)
	PublicURL(objectKey string) string
}

type StorageService struct {
	strg Storage
	repo MediaRepository
}

func NewStorageService(strg Storage, repo MediaRepository) *StorageService {
	return &StorageService{strg: strg, repo: repo}
}

type CreateMediaInput struct {
	Name string
	Type string
}

type CreateMediaOutput struct {
	ID        db.UUID
	UploadURL string
}

func (s *StorageService) CreateMedia(ctx context.Context, in CreateMediaInput) (CreateMediaOutput, error) {
	now := time.Now().UTC()
	objectKey := fmt.Sprintf("%s_%d", in.Name, now.UnixNano())
	media := &model.Media{
		ID:        db.NewUUID(),
		ObjectKey: objectKey,
		MimeType:  in.Type,
		SizeBytes: 0,
		Status:    "pending",
		Metadata:  "",
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := s.repo.Create(ctx, media); err != nil {
		return CreateMediaOutput{}, err
	}

	url, err := s.strg.GeneratePresignedUploadURL(ctx, objectKey, 1*time.Minute)
	if err != nil {
		return CreateMediaOutput{}, err
	}

	return CreateMediaOutput{
		ID:        media.ID,
		UploadURL: url,
	}, nil
}
