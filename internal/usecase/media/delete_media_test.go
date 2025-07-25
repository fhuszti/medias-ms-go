package media

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/fhuszti/medias-ms-go/internal/mock"
	"github.com/fhuszti/medias-ms-go/internal/model"
	msuuid "github.com/fhuszti/medias-ms-go/internal/uuid"
	"github.com/google/uuid"
)

func TestDeleteMedia_NotFound(t *testing.T) {
	repo := &mock.MediaRepo{GetByIDErr: sql.ErrNoRows}
	svc := NewMediaDeleter(repo, &mock.Cache{}, &mock.Storage{})

	id := msuuid.UUID(uuid.MustParse("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"))
	err := svc.DeleteMedia(context.Background(), id)
	if !errors.Is(err, ErrObjectNotFound) {
		t.Fatalf("expected ErrObjectNotFound, got %v", err)
	}
}

func TestDeleteMedia_GetByIDError(t *testing.T) {
	repo := &mock.MediaRepo{GetByIDErr: errors.New("db fail")}
	svc := NewMediaDeleter(repo, &mock.Cache{}, &mock.Storage{})

	id := msuuid.UUID(uuid.MustParse("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"))
	if err := svc.DeleteMedia(context.Background(), id); err == nil || err.Error() != "db fail" {
		t.Fatalf("expected db fail, got %v", err)
	}
}

func TestDeleteMedia_RemoveError(t *testing.T) {
	m := &model.Media{ID: msuuid.UUID(uuid.MustParse("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee")), Bucket: "images", ObjectKey: "k"}
	repo := &mock.MediaRepo{MediaOut: m}
	strg := &mock.Storage{RemoveErr: errors.New("remove fail")}
	svc := NewMediaDeleter(repo, &mock.Cache{}, strg)

	err := svc.DeleteMedia(context.Background(), m.ID)
	if err == nil || err.Error() != "remove fail" {
		t.Fatalf("expected remove fail, got %v", err)
	}
}

func TestDeleteMedia_DeleteError(t *testing.T) {
	m := &model.Media{ID: msuuid.UUID(uuid.MustParse("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee")), Bucket: "images", ObjectKey: "k"}
	repo := &mock.MediaRepo{MediaOut: m, DeleteErr: errors.New("delete fail")}
	strg := &mock.Storage{}
	svc := NewMediaDeleter(repo, &mock.Cache{}, strg)

	err := svc.DeleteMedia(context.Background(), m.ID)
	if err == nil || err.Error() != "delete fail" {
		t.Fatalf("expected delete fail, got %v", err)
	}
}

func TestDeleteMedia_Success(t *testing.T) {
	m := &model.Media{ID: msuuid.UUID(uuid.MustParse("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee")), Bucket: "images", ObjectKey: "k", Variants: model.Variants{{ObjectKey: "v1"}}}
	repo := &mock.MediaRepo{MediaOut: m}
	strg := &mock.Storage{}
	cache := &mock.Cache{}
	svc := NewMediaDeleter(repo, cache, strg)

	if err := svc.DeleteMedia(context.Background(), m.ID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strg.RemoveCalled {
		t.Error("expected RemoveFile to be called")
	}
	if !repo.DeleteCalled || repo.GotDeletedID != m.ID {
		t.Error("expected repo.Delete to be called with ID")
	}
	if !cache.DelMediaCalled {
		t.Error("expected cache delete to be called")
	}
	if !cache.DelEtagMediaCalled {
		t.Error("expected etag cache delete to be called")
	}
}
