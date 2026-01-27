package media

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/fhuszti/medias-ms-go/internal/logger"
	"github.com/fhuszti/medias-ms-go/internal/model"
	"github.com/fhuszti/medias-ms-go/internal/port"
)

type updateFinaliserSrv struct {
	repo  port.MediaRepository
	strg  port.Storage
	tasks port.TaskDispatcher
}

// compile-time check: *updateFinaliserSrv must satisfy port.UpdateFinaliser
var _ port.UpdateFinaliser = (*updateFinaliserSrv)(nil)

func NewUpdateFinaliser(repo port.MediaRepository, strg port.Storage, tasks port.TaskDispatcher) port.UpdateFinaliser {
	return &updateFinaliserSrv{repo, strg, tasks}
}

func (s *updateFinaliserSrv) FinaliseUpdate(ctx context.Context, in port.FinaliseUpdateInput) error {
	media, err := s.repo.GetByID(ctx, in.ID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrObjectNotFound
		}
		return err
	}
	if media.Status != model.MediaStatusCompleted {
		return errors.New("media status should be 'completed' to be updated")
	}

	nextVersion := media.Version + 1
	stagingKey := fmt.Sprintf("%s-v%d", media.ID, nextVersion)

	var finalErr error
	defer func() {
		if finalErr != nil {
			if err := cleanupStagingFile(stagingKey, s.strg); err != nil {
				logger.Errorf(ctx, "cleanup failed for file %q: %v", stagingKey, err)
			}
			if markErr := markMediaFailed(ctx, s.repo, media, finalErr.Error()); markErr != nil {
				logger.Errorf(ctx, "markAsFailed failed for file %q: %v", stagingKey, markErr)
			}
		}
	}()

	info, err := validateStagingFile(ctx, s.strg, stagingKey)
	if err != nil {
		finalErr = err
		return finalErr
	}

	newObjectKey, metadata, err := saveStagingFile(ctx, s.strg, stagingKey, in.DestBucket, info.SizeBytes, info.ContentType)
	if err != nil {
		finalErr = fmt.Errorf("move file %q from staging to bucket %q failed: %w", stagingKey, in.DestBucket, err)
		return finalErr
	}

	previousKey := media.ObjectKey
	previousBucket := media.Bucket
	previousVariants := append(model.Variants(nil), media.Variants...)

	updated := *media
	updated.ObjectKey = newObjectKey
	updated.Bucket = in.DestBucket
	updated.Version = nextVersion
	updated.Status = model.MediaStatusCompleted
	updated.SizeBytes = &info.SizeBytes
	updated.MimeType = &info.ContentType
	updated.Metadata = metadata
	updated.Optimised = false
	updated.FailureMessage = nil
	updated.Variants = model.Variants{}

	if err := s.repo.Update(ctx, &updated); err != nil {
		if remErr := s.strg.RemoveFile(ctx, in.DestBucket, newObjectKey); remErr != nil {
			logger.Warnf(ctx, "failed to remove file %q from bucket %q after update failure: %v", newObjectKey, in.DestBucket, remErr)
		}
		finalErr = fmt.Errorf("failed updating media: %w", err)
		return finalErr
	}

	if err := s.strg.RemoveFile(ctx, previousBucket, previousKey); err != nil {
		logger.Warnf(ctx, "failed to remove previous file %q from bucket %q: %v", previousKey, previousBucket, err)
	}
	for _, v := range previousVariants {
		if err := s.strg.RemoveFile(ctx, previousBucket, v.ObjectKey); err != nil {
			logger.Warnf(ctx, "failed to remove previous variant %q from bucket %q: %v", v.ObjectKey, previousBucket, err)
		}
	}

	*media = updated

	if err := s.tasks.EnqueueOptimiseMedia(ctx, media.ID); err != nil {
		logger.Warnf(ctx, "failed to enqueue optimise task for media #%s: %v", media.ID, err)
	}

	return nil
}
