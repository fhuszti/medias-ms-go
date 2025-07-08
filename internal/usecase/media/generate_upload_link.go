package media

import (
	"context"
	"time"

	"github.com/fhuszti/medias-ms-go/internal/model"
	"github.com/fhuszti/medias-ms-go/internal/port"
)

type uploadLinkGeneratorSrv struct {
	repo    port.MediaRepository
	strg    port.Storage
	genUUID port.UUIDGen
}

// compile-time check: *uploadLinkGeneratorSrv must satisfy port.UploadLinkGenerator
var _ port.UploadLinkGenerator = (*uploadLinkGeneratorSrv)(nil)

func NewUploadLinkGenerator(repo port.MediaRepository, strg port.Storage, genUUID port.UUIDGen) port.UploadLinkGenerator {
	return &uploadLinkGeneratorSrv{repo, strg, genUUID}
}

func (s *uploadLinkGeneratorSrv) GenerateUploadLink(ctx context.Context, in port.GenerateUploadLinkInput) (port.GenerateUploadLinkOutput, error) {
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
		return port.GenerateUploadLinkOutput{}, err
	}

	url, err := s.strg.GeneratePresignedUploadURL(ctx, "staging", objectKey, 5*time.Minute)
	if err != nil {
		return port.GenerateUploadLinkOutput{}, err
	}

	return port.GenerateUploadLinkOutput{
		ID:  media.ID,
		URL: url,
	}, nil
}
