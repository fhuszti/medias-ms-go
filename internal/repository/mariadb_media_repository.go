package repository

import (
	"context"
	"database/sql"
	"github.com/fhuszti/medias-ms-go/internal/model"
	"github.com/fhuszti/medias-ms-go/internal/service"
)

type MariaDBMediaRepository struct {
	db *sql.DB
}

// compile-time check: *MySQLMediaRepository must satisfy service.MediaRepository
var _ service.MediaRepository = (*MariaDBMediaRepository)(nil)

func NewMariaDBMediaRepository(db *sql.DB) *MariaDBMediaRepository {
	return &MariaDBMediaRepository{db: db}
}

func (r *MariaDBMediaRepository) Create(ctx context.Context, media *model.Media) error {
	const query = `
      INSERT INTO medias 
        (id, object_key, mime_type, size_bytes, status, metadata, created_at, updated_at)
      VALUES (?, ?, ?, ?, ?, ?, ?, ?)
    `
	_, err := r.db.ExecContext(ctx, query,
		media.ID, media.ObjectKey,
		media.MimeType, media.SizeBytes,
		media.Status, media.Metadata,
		media.CreatedAt, media.UpdatedAt,
	)
	if err != nil {
		return err
	}

	return nil
}
