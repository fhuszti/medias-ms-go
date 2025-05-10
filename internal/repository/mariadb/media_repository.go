package mariadb

import (
	"context"
	"database/sql"
	"github.com/fhuszti/medias-ms-go/internal/model"
	"github.com/fhuszti/medias-ms-go/internal/usecase/media"
	"log"
)

type MediaRepository struct {
	db *sql.DB
}

// compile-time check: *MediaRepository must satisfy media.Repository
var _ media.Repository = (*MediaRepository)(nil)

func NewMediaRepository(db *sql.DB) *MediaRepository {
	return &MediaRepository{db: db}
}

func (r *MediaRepository) Create(ctx context.Context, media *model.Media) error {
	log.Printf("creating database record for media '%s', at status '%s'...", media.ObjectKey, media.Status)
	const query = `
      INSERT INTO medias 
        (id, object_key, mime_type, size_bytes, status, failure_message, metadata)
      VALUES (?, ?, ?, ?, ?, ?, ?)
    `
	_, err := r.db.ExecContext(ctx, query,
		media.ID, media.ObjectKey,
		media.MimeType, media.SizeBytes,
		media.Status, media.FailureMessage,
		media.Metadata,
	)
	if err != nil {
		return err
	}

	return nil
}
