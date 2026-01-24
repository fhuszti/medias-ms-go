package media

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/fhuszti/medias-ms-go/internal/port"
)

type updateInitiatorSrv struct {
	repo port.MediaRepository
	strg port.Storage
}

// compile-time check: *updateInitiatorSrv must satisfy port.UpdateInitiator
var _ port.UpdateInitiator = (*updateInitiatorSrv)(nil)

func NewUpdateInitiator(repo port.MediaRepository, strg port.Storage) port.UpdateInitiator {
	return &updateInitiatorSrv{repo, strg}
}

func (s *updateInitiatorSrv) InitiateUpdate(ctx context.Context, in port.InitiateUpdateInput) (port.InitiateUpdateOutput, error) {
	media, err := s.repo.GetByID(ctx, in.ID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return port.InitiateUpdateOutput{}, ErrObjectNotFound
		}
		return port.InitiateUpdateOutput{}, err
	}

	nextVersion := media.Version + 1
	objectKey := fmt.Sprintf("%s-v%d", media.ID, nextVersion)

	url, err := s.strg.GeneratePresignedUploadURL(ctx, "staging", objectKey, 5*time.Minute)
	if err != nil {
		return port.InitiateUpdateOutput{}, err
	}

	return port.InitiateUpdateOutput{
		ID:  media.ID,
		URL: url,
	}, nil
}
