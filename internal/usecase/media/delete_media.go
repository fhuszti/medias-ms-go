package media

import (
	"context"
	"database/sql"
	"errors"
	"log"

	"github.com/fhuszti/medias-ms-go/internal/db"
	"github.com/fhuszti/medias-ms-go/internal/port"
)

type deleteMediaSrv struct {
	repo  port.MediaRepository
	cache port.Cache
	strg  port.Storage
}

// NewMediaDeleter constructs a MediaDeleter implementation.
func NewMediaDeleter(repo port.MediaRepository, cache port.Cache, strg port.Storage) port.MediaDeleter {
	return &deleteMediaSrv{repo: repo, cache: cache, strg: strg}
}

// DeleteMedia removes the file from storage, deletes DB record and clears the cache.
func (s *deleteMediaSrv) DeleteMedia(ctx context.Context, id db.UUID) error {
	media, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrObjectNotFound
		}
		return err
	}

	for _, v := range media.Variants {
		if err := s.strg.RemoveFile(ctx, media.Bucket, v.ObjectKey); err != nil {
			log.Printf("failed to remove variant %q: %v", v.ObjectKey, err)
		}
	}

	if err := s.strg.RemoveFile(ctx, media.Bucket, media.ObjectKey); err != nil {
		return err
	}

	if err := s.repo.Delete(ctx, media.ID); err != nil {
		return err
	}

	if err := s.cache.DeleteMediaDetails(ctx, media.ID); err != nil {
		log.Printf("failed deleting cache for media #%s: %v", media.ID, err)
	}
	if err := s.cache.DeleteEtagMediaDetails(ctx, media.ID); err != nil {
		log.Printf("failed deleting etag cache for media #%s: %v", media.ID, err)
	}

	return nil
}
