package media

import (
	"context"
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
	id := db.NewUUID()
	objectKey := id.String()
	media := &model.Media{
		ID:               id,
		ObjectKey:        objectKey,
		OriginalFilename: in.Name,
		Status:           model.MediaStatusPending,
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
