package media

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/fhuszti/medias-ms-go/internal/logger"
	"github.com/fhuszti/medias-ms-go/internal/model"
	"github.com/fhuszti/medias-ms-go/internal/port"
)

func cleanupStagingFile(objectKey string, strg port.Storage) error {
	if err := strg.RemoveFile(context.Background(), "staging", objectKey); err != nil {
		return err
	}
	return nil
}

func markMediaFailed(ctx context.Context, repo port.MediaRepository, media *model.Media, reason string) error {
	media.Status = model.MediaStatusFailed
	media.FailureMessage = &reason

	if err := repo.Update(ctx, media); err != nil {
		return err
	}
	return nil
}

func validateStagingFile(ctx context.Context, strg port.Storage, objectKey string) (port.FileInfo, error) {
	info, err := strg.StatFile(ctx, "staging", objectKey)
	if err != nil {
		if errors.Is(err, ErrObjectNotFound) {
			return port.FileInfo{}, fmt.Errorf("staging file %q not found", objectKey)
		}
		return port.FileInfo{}, fmt.Errorf("stats for file %q failed: %w", objectKey, err)
	}

	if info.SizeBytes < MinFileSize {
		return port.FileInfo{}, fmt.Errorf("file %q too small: %d bytes (min size: %d bytes)", objectKey, info.SizeBytes, MinFileSize)
	}
	if info.SizeBytes > MaxFileSize {
		return port.FileInfo{}, fmt.Errorf("file %q too large: %d bytes (max size: %d bytes)", objectKey, info.SizeBytes, MaxFileSize)
	}

	if !IsMimeTypeAllowed(info.ContentType) {
		return port.FileInfo{}, fmt.Errorf("unsupported mime-type %q for file %q", info.ContentType, objectKey)
	}

	return info, nil
}

func saveStagingFile(ctx context.Context, strg port.Storage, objectKey, destBucket string, size int64, contentType string) (string, model.Metadata, error) {
	file, err := strg.GetFile(ctx, "staging", objectKey)
	if err != nil {
		return "", model.Metadata{}, err
	}

	// Read metadata then reset reader position for saving.
	metadata, err := fillMetadata(contentType, file)
	if err != nil {
		return "", model.Metadata{}, fmt.Errorf("failed to fill metadata: %w", err)
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return "", model.Metadata{}, fmt.Errorf("failed to reset reader: %w", err)
	}
	defer func(file io.ReadSeekCloser) {
		if err := file.Close(); err != nil {
			logger.Warn(ctx, "failed to close reader")
		}
	}(file)

	ext, err := MimeTypeToExtension(contentType)
	if err != nil {
		return "", model.Metadata{}, err
	}
	newObjectKey := fmt.Sprintf("%s%s", objectKey, ext)

	if err := strg.SaveFile(
		ctx,
		destBucket,
		newObjectKey,
		file,
		size,
		map[string]string{
			"Content-Type": contentType,
		},
	); err != nil {
		return "", model.Metadata{}, err
	}

	if err := strg.RemoveFile(ctx, "staging", objectKey); err != nil {
		logger.Warnf(ctx, "failed to clean up file %q in staging: %v", objectKey, err)
	}

	return newObjectKey, metadata, nil
}
