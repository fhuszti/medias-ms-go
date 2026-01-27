package media

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/fhuszti/medias-ms-go/internal/model"
	"github.com/fhuszti/medias-ms-go/internal/port"

	"github.com/fhuszti/medias-ms-go/internal/logger"
)

type uploadFinaliserSrv struct {
	repo  port.MediaRepository
	strg  port.Storage
	tasks port.TaskDispatcher
}

// compile-time check: *uploadFinaliserSrv must satisfy port.UploadFinaliser
var _ port.UploadFinaliser = (*uploadFinaliserSrv)(nil)

func NewUploadFinaliser(repo port.MediaRepository, strg port.Storage, tasks port.TaskDispatcher) port.UploadFinaliser {
	return &uploadFinaliserSrv{repo, strg, tasks}
}

func (s *uploadFinaliserSrv) FinaliseUpload(ctx context.Context, in port.FinaliseUploadInput) error {
	media, err := s.repo.GetByID(ctx, in.ID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrObjectNotFound
		}
		return err
	}
	if media.Status == model.MediaStatusCompleted {
		return nil
	}
	if media.Status != model.MediaStatusPending {
		return errors.New("media status should be 'pending' to be finalised")
	}

	// Cleanup function
	var finalErr error
	defer func() {
		if finalErr != nil {
			if err := cleanupStagingFile(media.ObjectKey, s.strg); err != nil {
				logger.Errorf(ctx, "cleanup failed for file %q: %v", media.ObjectKey, err)
			}
			if markErr := markMediaFailed(ctx, s.repo, media, finalErr.Error()); markErr != nil {
				logger.Errorf(ctx, "markAsFailed failed for file %q: %v", media.ObjectKey, markErr)
			}
		}
	}()

	info, err := validateStagingFile(ctx, s.strg, media.ObjectKey)
	if err != nil {
		finalErr = err
		return finalErr
	}

	newObjectKey, metadata, err := saveStagingFile(ctx, s.strg, media.ObjectKey, in.DestBucket, info.SizeBytes, info.ContentType)
	if err != nil {
		finalErr = fmt.Errorf("move file %q from staging to bucket %q failed: %w", media.ObjectKey, in.DestBucket, err)
		return finalErr
	}

	updated := *media
	updated.ObjectKey = newObjectKey
	updated.Bucket = in.DestBucket
	updated.Status = model.MediaStatusCompleted
	updated.SizeBytes = &info.SizeBytes
	updated.MimeType = &info.ContentType
	updated.Metadata = metadata

	if err := s.repo.Update(ctx, &updated); err != nil {
		if remErr := s.strg.RemoveFile(ctx, in.DestBucket, newObjectKey); remErr != nil {
			logger.Warnf(ctx, "failed to remove file %q from bucket %q after update failure: %v", newObjectKey, in.DestBucket, remErr)
		}
		finalErr = fmt.Errorf("failed updating media: %w", err)
		return finalErr
	}

	*media = updated

	if err := s.tasks.EnqueueOptimiseMedia(ctx, media.ID); err != nil {
		logger.Warnf(ctx, "failed to enqueue optimise task for media #%s: %v", media.ID, err)
	}

	return nil
}
