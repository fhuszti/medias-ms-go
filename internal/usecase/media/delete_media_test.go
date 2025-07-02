package media

import (
	"context"
	"database/sql"
	"errors"
	"github.com/fhuszti/medias-ms-go/internal/mock"
	"testing"

	"github.com/fhuszti/medias-ms-go/internal/db"
	"github.com/fhuszti/medias-ms-go/internal/model"
	"github.com/google/uuid"
)

func TestDeleteMedia_NotFound(t *testing.T) {
	repo := &mock.MockMediaRepo{GetErr: sql.ErrNoRows}
	svc := NewMediaDeleter(repo, &mock.MockCache{}, &mock.MockStorage{})

	id := db.UUID(uuid.MustParse("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"))
	err := svc.DeleteMedia(context.Background(), DeleteMediaInput{ID: id})
	if !errors.Is(err, ErrObjectNotFound) {
		t.Fatalf("expected ErrObjectNotFound, got %v", err)
	}
}

func TestDeleteMedia_GetByIDError(t *testing.T) {
	repo := &mock.MockMediaRepo{GetErr: errors.New("db fail")}
	svc := NewMediaDeleter(repo, &mock.MockCache{}, &mock.MockStorage{})

	id := db.UUID(uuid.MustParse("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"))
	if err := svc.DeleteMedia(context.Background(), DeleteMediaInput{ID: id}); err == nil || err.Error() != "db fail" {
		t.Fatalf("expected db fail, got %v", err)
	}
}

func TestDeleteMedia_RemoveError(t *testing.T) {
	m := &model.Media{ID: db.UUID(uuid.MustParse("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee")), Bucket: "images", ObjectKey: "k"}
	repo := &mock.MockMediaRepo{MediaRecord: m}
	strg := &mock.MockStorage{RemoveErr: errors.New("remove fail")}
	svc := NewMediaDeleter(repo, &mock.MockCache{}, strg)

	err := svc.DeleteMedia(context.Background(), DeleteMediaInput{ID: m.ID})
	if err == nil || err.Error() != "remove fail" {
		t.Fatalf("expected remove fail, got %v", err)
	}
}

func TestDeleteMedia_DeleteError(t *testing.T) {
	m := &model.Media{ID: db.UUID(uuid.MustParse("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee")), Bucket: "images", ObjectKey: "k"}
	repo := &mock.MockMediaRepo{MediaRecord: m, DeleteErr: errors.New("delete fail")}
	strg := &mock.MockStorage{}
	svc := NewMediaDeleter(repo, &mock.MockCache{}, strg)

	err := svc.DeleteMedia(context.Background(), DeleteMediaInput{ID: m.ID})
	if err == nil || err.Error() != "delete fail" {
		t.Fatalf("expected delete fail, got %v", err)
	}
}

func TestDeleteMedia_Success(t *testing.T) {
	m := &model.Media{ID: db.UUID(uuid.MustParse("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee")), Bucket: "images", ObjectKey: "k", Variants: model.Variants{{ObjectKey: "v1"}}}
	repo := &mock.MockMediaRepo{MediaRecord: m}
	strg := &mock.MockStorage{}
	cache := &mock.MockCache{}
	svc := NewMediaDeleter(repo, cache, strg)

	if err := svc.DeleteMedia(context.Background(), DeleteMediaInput{ID: m.ID}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strg.RemoveCalled {
		t.Error("expected RemoveFile to be called")
	}
	if !repo.DeleteCalled || repo.DeletedID != m.ID {
		t.Error("expected repo.Delete to be called with ID")
	}
	if !cache.DelMediaCalled {
		t.Error("expected cache delete to be called")
	}
	if !cache.DelEtagCalled {
		t.Error("expected etag cache delete to be called")
	}
}
