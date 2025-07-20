package mariadb

import (
	"context"
	"database/sql"
	"log"
	"time"

	"github.com/fhuszti/medias-ms-go/internal/model"
	"github.com/fhuszti/medias-ms-go/internal/port"
	msuuid "github.com/fhuszti/medias-ms-go/internal/uuid"
)

type MediaRepository struct {
	db *sql.DB
}

// compile-time check: *MediaRepository must satisfy port.MediaRepository
var _ port.MediaRepository = (*MediaRepository)(nil)

func NewMediaRepository(db *sql.DB) *MediaRepository {
	return &MediaRepository{db: db}
}

func (r *MediaRepository) GetByID(ctx context.Context, ID msuuid.UUID) (*model.Media, error) {
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

func (r *MediaRepository) Delete(ctx context.Context, ID msuuid.UUID) error {
	log.Printf("deleting media #%s from the database...", ID)

	const query = `DELETE FROM medias WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, ID)
	return err
}

func (r *MediaRepository) ListUnoptimisedCompletedBefore(ctx context.Context, before time.Time) ([]msuuid.UUID, error) {
	log.Printf("fetching medias to reoptimise before %s...", before)

	const query = `
      SELECT id FROM medias
      WHERE status = ? AND optimised = FALSE AND created_at <= ?
    `
	rows, err := r.db.QueryContext(ctx, query, model.MediaStatusCompleted, before)
	if err != nil {
		return nil, err
	}
	defer func() {
		if cerr := rows.Close(); cerr != nil {
			log.Printf("rows close error: %v", cerr)
		}
	}()

	var ids []msuuid.UUID
	for rows.Next() {
		var id msuuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return ids, nil
}

func (r *MediaRepository) ListOptimisedImagesNoVariantsBefore(ctx context.Context, before time.Time) ([]msuuid.UUID, error) {
	log.Printf("fetching images to resize before %s...", before)

	const query = `
      SELECT id FROM medias
      WHERE status = ?
        AND optimised = TRUE
        AND JSON_LENGTH(variants) = 0
        AND created_at <= ?
        AND mime_type LIKE 'image/%'
    `
	rows, err := r.db.QueryContext(ctx, query, model.MediaStatusCompleted, before)
	if err != nil {
		return nil, err
	}
	defer func() {
		if cerr := rows.Close(); cerr != nil {
			log.Printf("rows close error: %v", cerr)
		}
	}()

	var ids []msuuid.UUID
	for rows.Next() {
		var id msuuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return ids, nil
}
