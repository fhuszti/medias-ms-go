package media

import (
	"context"
	"fmt"
	"time"

	"github.com/fhuszti/medias-ms-go/internal/model"
	"github.com/fhuszti/medias-ms-go/internal/port"
)

type uploadInitiatorSrv struct {
	repo    port.MediaRepository
	strg    port.Storage
	genUUID port.UUIDGen
}

// compile-time check: *uploadInitiatorSrv must satisfy port.UploadInitiator
var _ port.UploadInitiator = (*uploadInitiatorSrv)(nil)

func NewUploadInitiator(repo port.MediaRepository, strg port.Storage, genUUID port.UUIDGen) port.UploadInitiator {
	return &uploadInitiatorSrv{repo, strg, genUUID}
}

func (s *uploadInitiatorSrv) InitiateUpload(ctx context.Context, in port.InitiateUploadInput) (port.InitiateUploadOutput, error) {
	id := s.genUUID()
	objectKey := fmt.Sprintf("%s-v1", id)
	media := &model.Media{
		ID:               id,
		ObjectKey:        objectKey,
		Bucket:           "staging",
		OriginalFilename: in.Name,
		Version:          1,
		Status:           model.MediaStatusPending,
		Metadata:         model.Metadata{},
		Variants:         model.Variants{},
	}

	if err := s.repo.Create(ctx, media); err != nil {
		return port.InitiateUploadOutput{}, err
	}

	url, err := s.strg.GeneratePresignedUploadURL(ctx, "staging", objectKey, 5*time.Minute)
	if err != nil {
		return port.InitiateUploadOutput{}, err
	}

	return port.InitiateUploadOutput{
		ID:  media.ID,
		URL: url,
	}, nil
}
