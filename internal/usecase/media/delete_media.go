package media

import (
	"context"
	"database/sql"
	"errors"
	"log"

	"github.com/fhuszti/medias-ms-go/internal/db"
)

// Deleter deletes a media and its file.
type Deleter interface {
	DeleteMedia(ctx context.Context, in DeleteMediaInput) error
}

type deleteMediaSrv struct {
	repo  Repository
	cache Cache
	strg  Storage
}

// NewMediaDeleter constructs a Deleter implementation.
func NewMediaDeleter(repo Repository, cache Cache, strg Storage) Deleter {
	return &deleteMediaSrv{repo: repo, cache: cache, strg: strg}
}

// DeleteMediaInput represents the input for deleting a media.
type DeleteMediaInput struct {
	ID db.UUID
}

// DeleteMedia removes the file from storage, deletes DB record and clears cache.
func (s *deleteMediaSrv) DeleteMedia(ctx context.Context, in DeleteMediaInput) error {
	media, err := s.repo.GetByID(ctx, in.ID)
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

	return nil
}
