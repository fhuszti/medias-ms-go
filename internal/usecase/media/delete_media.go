package media

import (
	"context"
	"database/sql"
	"errors"

	"github.com/fhuszti/medias-ms-go/internal/port"
	msuuid "github.com/fhuszti/medias-ms-go/internal/uuid"

	"github.com/fhuszti/medias-ms-go/internal/logger"
)

type deleteMediaSrv struct {
	repo  port.MediaRepository
	cache port.Cache
	strg  port.Storage
}

// compile-time check: *deleteMediaSrv must satisfy port.MediaDeleter
var _ port.MediaDeleter = (*deleteMediaSrv)(nil)

// NewMediaDeleter constructs a MediaDeleter implementation.
func NewMediaDeleter(repo port.MediaRepository, cache port.Cache, strg port.Storage) port.MediaDeleter {
	return &deleteMediaSrv{repo: repo, cache: cache, strg: strg}
}

// DeleteMedia removes the file from storage, deletes DB record and clears the cache.
func (s *deleteMediaSrv) DeleteMedia(ctx context.Context, id msuuid.UUID) error {
	media, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrObjectNotFound
		}
		return err
	}

	for _, v := range media.Variants {
		if err := s.strg.RemoveFile(ctx, media.Bucket, v.ObjectKey); err != nil {
			logger.Warnf(ctx, "failed to remove variant %q: %v", v.ObjectKey, err)
		}
	}

	if err := s.strg.RemoveFile(ctx, media.Bucket, media.ObjectKey); err != nil {
		return err
	}

	if err := s.repo.Delete(ctx, media.ID); err != nil {
		return err
	}

	if err := s.cache.DeleteMediaDetails(ctx, media.ID); err != nil {
		logger.Warnf(ctx, "failed deleting cache for media #%s: %v", media.ID, err)
	}
	if err := s.cache.DeleteEtagMediaDetails(ctx, media.ID); err != nil {
		logger.Warnf(ctx, "failed deleting etag cache for media #%s: %v", media.ID, err)
	}

	return nil
}
