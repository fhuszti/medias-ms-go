package media

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"

	"github.com/fhuszti/medias-ms-go/internal/db"
	"github.com/fhuszti/medias-ms-go/internal/model"
)

type UploadFinaliser interface {
	FinaliseUpload(ctx context.Context, in FinaliseUploadInput) (*model.Media, error)
}

type uploadFinaliserSrv struct {
	repo          Repository
	stagingStrg   Storage
	getDestBucket StorageGetter
}

func NewUploadFinaliser(repo Repository, stagingStrg Storage, getDestBucket StorageGetter) UploadFinaliser {
	return &uploadFinaliserSrv{repo: repo, stagingStrg: stagingStrg, getDestBucket: getDestBucket}
}

type FinaliseUploadInput struct {
	ID         db.UUID
	DestBucket string
}

func (s *uploadFinaliserSrv) FinaliseUpload(ctx context.Context, in FinaliseUploadInput) (*model.Media, error) {
	media, err := s.repo.GetByID(ctx, in.ID)
	if err != nil {
		return nil, err
	}
	if media.Status == model.MediaStatusCompleted {
		return media, nil
	}
	if media.Status != model.MediaStatusPending {
		return nil, errors.New("media status should be 'pending' to be finalised")
	}

	// Cleanup function
	var finalErr error
	defer func() {
		if finalErr != nil {
			if err := s.cleanupFile(media.ObjectKey); err != nil {
				log.Printf("cleanup failed for file %q: %v", media.ObjectKey, err)
			}
			if markErr := s.markAsFailed(ctx, media, finalErr.Error()); markErr != nil {
				log.Printf("markAsFailed failed for file %q: %v", media.ObjectKey, markErr)
			}
		}
	}()

	info, err := s.stagingStrg.StatFile(ctx, media.ObjectKey)
	if err != nil {
		if errors.Is(err, ErrObjectNotFound) {
			finalErr = fmt.Errorf("staging file %q not found", media.ObjectKey)
		} else {
			finalErr = fmt.Errorf("stats for file %q failed: %w", media.ObjectKey, err)
		}
		return nil, finalErr
	}

	if info.SizeBytes < MinFileSize {
		finalErr = fmt.Errorf("file %q too small: %d bytes", media.ObjectKey, info.SizeBytes)
		return nil, finalErr
	}
	if info.SizeBytes > MaxFileSize {
		finalErr = fmt.Errorf("file %q too large: %d bytes", media.ObjectKey, info.SizeBytes)
		return nil, finalErr
	}

	if !IsMimeTypeAllowed(info.ContentType) {
		finalErr = fmt.Errorf("unsupported mime-type %q for file %q", info.ContentType, media.ObjectKey)
		return nil, finalErr
	}

	if err := s.moveFile(ctx, media, info.SizeBytes, info.ContentType, in.DestBucket); err != nil {
		finalErr = fmt.Errorf("move file %q from staging to bucket %q failed: %w", media.ObjectKey, in.DestBucket, err)
		return nil, finalErr
	}

	media.Status = model.MediaStatusCompleted
	media.SizeBytes = &info.SizeBytes
	media.MimeType = &info.ContentType
	if err := s.repo.Update(ctx, media); err != nil {
		finalErr = fmt.Errorf("failed updating media after finalising the upload: %w", err)
		return nil, finalErr
	}

	return media, nil
}

func (s *uploadFinaliserSrv) cleanupFile(objectKey string) error {
	if err := s.stagingStrg.RemoveFile(context.Background(), objectKey); err != nil {
		return err
	}
	return nil
}

func (s *uploadFinaliserSrv) markAsFailed(ctx context.Context, media *model.Media, reason string) error {
	media.Status = model.MediaStatusFailed
	media.FailureMessage = &reason

	if err := s.repo.Update(ctx, media); err != nil {
		return err
	}
	return nil
}

func (s *uploadFinaliserSrv) moveFile(ctx context.Context, media *model.Media, size int64, contentType string, destBucket string) error {
	destStrg, err := s.getDestBucket(destBucket)
	if err != nil {
		return fmt.Errorf("unknown destination bucket %q: %w", destBucket, err)
	}

	objReader, err := s.stagingStrg.GetFile(ctx, media.ObjectKey)
	if err != nil {
		return err
	}
	defer func(objReader io.ReadCloser) {
		if err := objReader.Close(); err != nil {
			log.Printf("failed to close reader")
		}
	}(objReader)

	if err := destStrg.SaveFile(
		ctx,
		media.ObjectKey,
		objReader,
		size,
		map[string]string{
			"Content-Type": contentType,
		},
	); err != nil {
		return err
	}

	if err := s.stagingStrg.RemoveFile(ctx, media.ObjectKey); err != nil {
		log.Printf("failed to clean up file %q in staging: %v", media.ObjectKey, err)
	}

	return nil
}
