package media

import (
	"context"
	"time"

	"github.com/fhuszti/medias-ms-go/internal/db"
	"github.com/fhuszti/medias-ms-go/internal/model"
	"github.com/fhuszti/medias-ms-go/internal/port"
)

type UUIDGen func() db.UUID

type UploadLinkGenerator interface {
	GenerateUploadLink(ctx context.Context, in GenerateUploadLinkInput) (GenerateUploadLinkOutput, error)
}

type uploadLinkGeneratorSrv struct {
	repo    port.MediaRepository
	strg    port.Storage
	genUUID UUIDGen
}

func NewUploadLinkGenerator(repo port.MediaRepository, strg port.Storage, genUUID UUIDGen) UploadLinkGenerator {
	return &uploadLinkGeneratorSrv{repo, strg, genUUID}
}

type GenerateUploadLinkInput struct {
	Name string
}

type GenerateUploadLinkOutput struct {
	ID  db.UUID `json:"id"`
	URL string  `json:"url"`
}

func (s *uploadLinkGeneratorSrv) GenerateUploadLink(ctx context.Context, in GenerateUploadLinkInput) (GenerateUploadLinkOutput, error) {
	id := s.genUUID()
	objectKey := id.String()
	media := &model.Media{
		ID:               id,
		ObjectKey:        objectKey,
		Bucket:           "staging",
		OriginalFilename: in.Name,
		Status:           model.MediaStatusPending,
		Metadata:         model.Metadata{},
		Variants:         model.Variants{},
	}

	if err := s.repo.Create(ctx, media); err != nil {
		return GenerateUploadLinkOutput{}, err
	}

	url, err := s.strg.GeneratePresignedUploadURL(ctx, "staging", objectKey, 5*time.Minute)
	if err != nil {
		return GenerateUploadLinkOutput{}, err
	}

	return GenerateUploadLinkOutput{
		ID:  media.ID,
		URL: url,
	}, nil
}
