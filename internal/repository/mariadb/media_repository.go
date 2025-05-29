package mariadb

import (
	"context"
	"database/sql"
	"github.com/fhuszti/medias-ms-go/internal/db"
	"github.com/fhuszti/medias-ms-go/internal/model"
	mediaService "github.com/fhuszti/medias-ms-go/internal/usecase/media"
	"log"
)

type MediaRepository struct {
	db *sql.DB
}

// compile-time check: *MediaRepository must satisfy media.Repository
var _ mediaService.Repository = (*MediaRepository)(nil)

func NewMediaRepository(db *sql.DB) *MediaRepository {
	return &MediaRepository{db: db}
}

func (r *MediaRepository) Create(ctx context.Context, media *model.Media) error {
	log.Printf("creating database record for media #%s, at status %q...", media.ID, media.Status)

	const query = `
      INSERT INTO medias 
        (id, object_key, bucket, original_filename, mime_type, size_bytes, status, optimised, failure_message, metadata, variants)
      VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
    `
	_, err := r.db.ExecContext(ctx, query,
		media.ID, media.ObjectKey, media.Bucket,
		media.OriginalFilename, media.MimeType,
		media.SizeBytes, media.Status, media.Optimised,
		media.FailureMessage, media.Metadata, media.Variants,
	)
	if err != nil {
		return err
	}

	return nil
}

func (r *MediaRepository) Update(ctx context.Context, media *model.Media) error {
	log.Printf("updating database record for media #%s, with status %q...", media.ID, media.Status)

	const query = `
      UPDATE medias
      SET
        object_key      = ?,
        bucket     		= ?,
        mime_type       = ?,
        size_bytes      = ?,
        status          = ?,
        optimised       = ?,
        failure_message = ?,
        metadata        = ?,
        variants        = ?
      WHERE id = ?
    `
	_, err := r.db.ExecContext(ctx, query,
		media.ObjectKey,
		media.Bucket,
		media.MimeType,
		media.SizeBytes,
		media.Status,
		media.Optimised,
		media.FailureMessage,
		media.Metadata,
		media.Variants,
		media.ID, // WHERE clause
	)
	if err != nil {
		return err
	}

	return nil
}

func (r *MediaRepository) GetByID(ctx context.Context, ID db.UUID) (*model.Media, error) {
	log.Printf("fetching media #%s from the database...", ID)

	const query = `
      SELECT id, object_key, bucket, original_filename, mime_type, size_bytes, status, optimised, failure_message, metadata, variants, created_at, updated_at
      FROM medias
      WHERE id = ?
    `
	row := r.db.QueryRowContext(ctx, query, ID)
	var media model.Media
	if err := row.Scan(
		&media.ID, &media.ObjectKey, &media.Bucket,
		&media.OriginalFilename, &media.MimeType,
		&media.SizeBytes, &media.Status, &media.Optimised,
		&media.FailureMessage, &media.Metadata, &media.Variants,
		&media.CreatedAt, &media.UpdatedAt,
	); err != nil {
		return nil, err
	}

	return &media, nil
}
