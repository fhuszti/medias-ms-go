package media

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/fhuszti/medias-ms-go/internal/db"
	"github.com/fhuszti/medias-ms-go/internal/model"
	"github.com/google/uuid"
)

func TestDeleteMedia_NotFound(t *testing.T) {
	repo := &mockRepo{getErr: sql.ErrNoRows}
	svc := NewMediaDeleter(repo, &mockCache{}, &mockStorage{})

	id := db.UUID(uuid.MustParse("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"))
	err := svc.DeleteMedia(context.Background(), DeleteMediaInput{ID: id})
	if !errors.Is(err, ErrObjectNotFound) {
		t.Fatalf("expected ErrObjectNotFound, got %v", err)
	}
}

func TestDeleteMedia_GetByIDError(t *testing.T) {
	repo := &mockRepo{getErr: errors.New("db fail")}
	svc := NewMediaDeleter(repo, &mockCache{}, &mockStorage{})

	id := db.UUID(uuid.MustParse("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"))
	if err := svc.DeleteMedia(context.Background(), DeleteMediaInput{ID: id}); err == nil || err.Error() != "db fail" {
		t.Fatalf("expected db fail, got %v", err)
	}
}

func TestDeleteMedia_RemoveError(t *testing.T) {
	m := &model.Media{ID: db.UUID(uuid.MustParse("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee")), Bucket: "images", ObjectKey: "k"}
	repo := &mockRepo{mediaRecord: m}
	strg := &mockStorage{removeErr: errors.New("remove fail")}
	svc := NewMediaDeleter(repo, &mockCache{}, strg)

	err := svc.DeleteMedia(context.Background(), DeleteMediaInput{ID: m.ID})
	if err == nil || err.Error() != "remove fail" {
		t.Fatalf("expected remove fail, got %v", err)
	}
}

func TestDeleteMedia_DeleteError(t *testing.T) {
	m := &model.Media{ID: db.UUID(uuid.MustParse("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee")), Bucket: "images", ObjectKey: "k"}
	repo := &mockRepo{mediaRecord: m, deleteErr: errors.New("delete fail")}
	strg := &mockStorage{}
	svc := NewMediaDeleter(repo, &mockCache{}, strg)

	err := svc.DeleteMedia(context.Background(), DeleteMediaInput{ID: m.ID})
	if err == nil || err.Error() != "delete fail" {
		t.Fatalf("expected delete fail, got %v", err)
	}
}

func TestDeleteMedia_Success(t *testing.T) {
	m := &model.Media{ID: db.UUID(uuid.MustParse("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee")), Bucket: "images", ObjectKey: "k", Variants: model.Variants{{ObjectKey: "v1"}}}
	repo := &mockRepo{mediaRecord: m}
	strg := &mockStorage{}
	cache := &mockCache{}
	svc := NewMediaDeleter(repo, cache, strg)

	if err := svc.DeleteMedia(context.Background(), DeleteMediaInput{ID: m.ID}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strg.removeCalled {
		t.Error("expected RemoveFile to be called")
	}
	if !repo.deleteCalled || repo.deletedID != m.ID {
		t.Error("expected repo.Delete to be called with ID")
	}
	if !cache.delMediaCalled {
		t.Error("expected cache delete to be called")
	}
}
