package media

import (
	"context"
	"fmt"
	"time"

	"github.com/fhuszti/medias-ms-go/internal/db"
	"github.com/fhuszti/medias-ms-go/internal/model"
)

type UploadLinkGenerator interface {
	GenerateUploadLink(ctx context.Context, in GenerateUploadLinkInput) (GenerateUploadLinkOutput, error)
}

type uploadLinkGeneratorSrv struct {
	repo Repository
	strg Storage
}

func NewUploadLinkGenerator(repo Repository, strg Storage) UploadLinkGenerator {
	return &uploadLinkGeneratorSrv{repo: repo, strg: strg}
}

type GenerateUploadLinkInput struct {
	Name string
}

type GenerateUploadLinkOutput struct {
	ID  db.UUID `json:"id"`
	URL string  `json:"url"`
}

func (s *uploadLinkGeneratorSrv) GenerateUploadLink(ctx context.Context, in GenerateUploadLinkInput) (GenerateUploadLinkOutput, error) {
	now := time.Now().UTC()
	objectKey := fmt.Sprintf("%s_%d", in.Name, now.UnixNano())
	media := &model.Media{
		ID:        db.NewUUID(),
		ObjectKey: objectKey,
		Status:    model.MediaStatusPending,
	}

	if err := s.repo.Create(ctx, media); err != nil {
		return GenerateUploadLinkOutput{}, err
	}

	url, err := s.strg.GeneratePresignedUploadURL(ctx, objectKey, 5*time.Minute)
	if err != nil {
		return GenerateUploadLinkOutput{}, err
	}

	return GenerateUploadLinkOutput{
		ID:  media.ID,
		URL: url,
	}, nil
}
