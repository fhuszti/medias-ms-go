package media

import (
	"context"
	"fmt"
	"time"

	"github.com/fhuszti/medias-ms-go/internal/db"
	"github.com/fhuszti/medias-ms-go/internal/model"
)

type UploadLinkGenerator interface {
	GenerateUploadLink(ctx context.Context, in GenerateUploadLinkInput) (string, error)
}

type service struct {
	repo Repository
	strg Storage
}

func NewUploadLinkGenerator(repo Repository, strg Storage) UploadLinkGenerator {
	return &service{repo: repo, strg: strg}
}

type GenerateUploadLinkInput struct {
	Name string
	Type string
}

func (s *service) GenerateUploadLink(ctx context.Context, in GenerateUploadLinkInput) (string, error) {
	now := time.Now().UTC()
	objectKey := fmt.Sprintf("%s_%d", in.Name, now.UnixNano())
	media := &model.Media{
		ID:        db.NewUUID(),
		ObjectKey: objectKey,
		MimeType:  in.Type,
		Status:    model.MediaStatusPending,
	}

	if err := s.repo.Create(ctx, media); err != nil {
		return "", err
	}

	url, err := s.strg.GeneratePresignedUploadURL(ctx, objectKey, 5*time.Minute)
	if err != nil {
		return "", err
	}

	return url, nil
}
