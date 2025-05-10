package mariadb

import (
	"context"
	"errors"
	"github.com/google/uuid"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/fhuszti/medias-ms-go/internal/db"
	"github.com/fhuszti/medias-ms-go/internal/model"
)

func TestMediaRepository_Create_Success(t *testing.T) {
	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("unexpected error when opening stub database: %s", err)
	}
	defer func() { _ = sqlDB.Close() }()

	repo := NewMediaRepository(sqlDB)

	mockID := db.UUID(uuid.MustParse("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"))
	size := 12345
	failure := "oops happened"
	m := &model.Media{
		ID:             mockID,
		ObjectKey:      "mykey",
		MimeType:       "image/png",
		SizeBytes:      &size,
		Status:         model.MediaStatusPending,
		FailureMessage: &failure,
		Metadata:       nil,
	}

	mock.ExpectExec(regexp.QuoteMeta(`
      INSERT INTO medias 
        (id, object_key, mime_type, size_bytes, status, failure_message, metadata)
      VALUES (?, ?, ?, ?, ?, ?, ?)
    `)).
		WithArgs(
			m.ID,
			sqlmock.AnyArg(), // ObjectKey
			m.MimeType,
			m.SizeBytes,
			m.Status,
			m.FailureMessage,
			m.Metadata,
		).
		WillReturnResult(sqlmock.NewResult(1, 1))

	if err := repo.Create(context.Background(), m); err != nil {
		t.Errorf("Create() returned unexpected error: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestMediaRepository_Create_ExecError(t *testing.T) {
	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("unexpected error when opening stub database: %s", err)
	}
	defer func() { _ = sqlDB.Close() }()

	repo := NewMediaRepository(sqlDB)

	mockID := db.UUID(uuid.MustParse("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"))
	size := 0
	failure := ""
	m := &model.Media{
		ID:             mockID,
		ObjectKey:      "otherkey",
		MimeType:       "application/json",
		SizeBytes:      &size,
		Status:         model.MediaStatusPending,
		FailureMessage: &failure,
		Metadata:       nil,
	}

	mock.ExpectExec("INSERT INTO medias").
		WithArgs(
			m.ID,
			sqlmock.AnyArg(),
			m.MimeType,
			m.SizeBytes,
			m.Status,
			m.FailureMessage,
			m.Metadata,
		).
		WillReturnError(errors.New("db.Exec failed"))

	err = repo.Create(context.Background(), m)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err.Error() != "db.Exec failed" {
		t.Errorf("expected 'db.Exec failed', got %q", err.Error())
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}
