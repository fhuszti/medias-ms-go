package media

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/fhuszti/medias-ms-go/internal/db"
	"github.com/fhuszti/medias-ms-go/internal/model"
	"github.com/minio/minio-go/v7"
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

	if media.Status != model.MediaStatusPending {
		return nil, errors.New("media status should be 'pending' to be finalised")
	}

	exists, err := s.stagingStrg.FileExists(ctx, media.ObjectKey)
	if err != nil {
		return nil, err
	}
	if !exists {
		errStr := fmt.Sprintf("file '%s' does not exist in staging", media.ObjectKey)
		if err := s.markAsFailed(ctx, media, errStr); err != nil {
			return nil, err
		}
		return nil, errors.New(errStr)
	}

	info, err := s.stagingStrg.StatFile(ctx, media.ObjectKey)
	if err != nil {
		errStr := fmt.Sprintf("could not get stats of file '%s' in staging", media.ObjectKey)
		if err := s.cleanupFile(media.ObjectKey); err != nil {
			return nil, err
		}
		if err := s.markAsFailed(ctx, media, errStr); err != nil {
			return nil, err
		}
		return nil, err
	}

	intSize := int(info.Size)
	if intSize > MaxFileSize {
		errStr := fmt.Sprintf("file '%s' is too large (%d bytes)", media.ObjectKey, info.Size)
		if err := s.cleanupFile(media.ObjectKey); err != nil {
			return nil, err
		}
		if err := s.markAsFailed(ctx, media, errStr); err != nil {
			return nil, err
		}
		return nil, errors.New(errStr)
	}

	if !IsMimeTypeAllowed(info.ContentType) {
		errStr := fmt.Sprintf("unsupported content type for file '%s': %s", media.ObjectKey, info.ContentType)
		if err := s.cleanupFile(media.ObjectKey); err != nil {
			return nil, err
		}
		if err := s.markAsFailed(ctx, media, errStr); err != nil {
			return nil, err
		}
		return nil, errors.New(errStr)
	}

	if err := s.moveFile(ctx, media, info.Size, info.ContentType, in.DestBucket); err != nil {
		errStr := fmt.Sprintf("failed to move file '%s' from staging to destination bucket", media.ObjectKey)
		if err := s.cleanupFile(media.ObjectKey); err != nil {
			return nil, err
		}
		if err := s.markAsFailed(ctx, media, errStr); err != nil {
			return nil, err
		}
		return nil, err
	}

	media.Status = model.MediaStatusCompleted
	media.SizeBytes = &intSize
	media.MimeType = &info.ContentType
	if err := s.repo.Update(ctx, media); err != nil {
		return nil, err
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
		return fmt.Errorf("unknown destination bucket %s: %w", destBucket, err)
	}

	objReader, err := s.stagingStrg.GetFile(ctx, media.ObjectKey)
	if err != nil {
		return err
	}
	defer func(objReader *minio.Object) {
		if err := objReader.Close(); err != nil {
			log.Printf("failed to close reader")
		}
	}(objReader)

	if _, err := destStrg.SaveFile(
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
		log.Printf("failed to clean up file '%s' in staging: %v", media.ObjectKey, err)
	}

	return nil
}
