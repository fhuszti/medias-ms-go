package media

import (
	"context"
	"database/sql"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/fhuszti/medias-ms-go/internal/mock"
	"github.com/fhuszti/medias-ms-go/internal/model"
	"github.com/fhuszti/medias-ms-go/internal/port"
	msuuid "github.com/fhuszti/medias-ms-go/internal/uuid"
	"github.com/google/uuid"
)

func TestInitiateUpdate_Success(t *testing.T) {
	mockID := msuuid.UUID(uuid.MustParse("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"))
	existing := &model.Media{
		ID:               mockID,
		ObjectKey:        "old-key.webp",
		Bucket:           "images",
		OriginalFilename: "old.webp",
		Version:          2,
		Status:           model.MediaStatusCompleted,
		Optimised:        true,
		Metadata:         model.Metadata{Width: 10, Height: 20},
		Variants:         model.Variants{},
	}

	repo := &mock.MediaRepo{MediaOut: existing}
	strg := &mock.Storage{}
	svc := NewUpdateInitiator(repo, strg)

	in := port.InitiateUpdateInput{
		ID:   mockID,
		Name: "new-file.png",
	}
	out, err := svc.InitiateUpdate(context.Background(), in)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if out.ID != mockID {
		t.Errorf("expected ID %q, got %q", mockID, out.ID)
	}
	if out.URL != "https://example.com/upload" {
		t.Errorf("expected url %q, got %q", "https://example.com/upload", out.URL)
	}

	if !repo.GetByIDCalled {
		t.Fatal("expected repo.GetByID to be called")
	}
	if !repo.UpdateCalled {
		t.Fatal("expected repo.Update to be called")
	}

	updated := repo.GotUpdated
	if updated == nil {
		t.Fatal("expected repo.Update to be called with media")
	}
	if updated.Version != 3 {
		t.Errorf("expected Version 3, got %d", updated.Version)
	}
	if updated.ObjectKey != mockID.String()+"-v3" {
		t.Errorf("expected ObjectKey %q, got %q", mockID.String()+"-v3", updated.ObjectKey)
	}
	if updated.Bucket != "staging" {
		t.Errorf("expected Bucket 'staging', got %q", updated.Bucket)
	}
	if updated.OriginalFilename != in.Name {
		t.Errorf("expected OriginalFilename %q, got %q", in.Name, updated.OriginalFilename)
	}
	if updated.Status != model.MediaStatusPending {
		t.Errorf("expected Status Pending, got %v", updated.Status)
	}
	if updated.Optimised {
		t.Errorf("expected Optimised false, got true")
	}
	if updated.MimeType != nil {
		t.Errorf("expected MimeType nil, got %v", *updated.MimeType)
	}
	if updated.SizeBytes != nil {
		t.Errorf("expected SizeBytes nil, got %v", *updated.SizeBytes)
	}
	if updated.FailureMessage != nil {
		t.Errorf("expected FailureMessage nil, got %v", *updated.FailureMessage)
	}
	if updated.Metadata != (model.Metadata{}) {
		t.Errorf("expected empty Metadata struct, got %+v", updated.Metadata)
	}
	if !reflect.DeepEqual(updated.Variants, model.Variants{}) {
		t.Errorf("expected empty Variants, got %+v", updated.Variants)
	}

	if !strg.InitiateUploadCalled {
		t.Error("expected strg.GeneratePresignedUploadURL to be called")
	}
	if strg.ObjectKey != updated.ObjectKey {
		t.Errorf("strg called with key %q, want %q", strg.ObjectKey, updated.ObjectKey)
	}
	if strg.TTL != 5*time.Minute {
		t.Errorf("strg called with TTL %v, want %v", strg.TTL, 5*time.Minute)
	}
}

func TestInitiateUpdate_NotFound(t *testing.T) {
	repo := &mock.MediaRepo{GetByIDErr: sql.ErrNoRows}
	strg := &mock.Storage{}
	svc := NewUpdateInitiator(repo, strg)

	_, err := svc.InitiateUpdate(context.Background(), port.InitiateUpdateInput{
		ID:   msuuid.UUID(uuid.Nil),
		Name: "file.png",
	})
	if !errors.Is(err, ErrObjectNotFound) {
		t.Fatalf("expected ErrObjectNotFound, got %v", err)
	}
	if repo.UpdateCalled {
		t.Error("did not expect repo.Update to be called")
	}
	if strg.InitiateUploadCalled {
		t.Error("did not expect strg.GeneratePresignedUploadURL to be called")
	}
}

func TestInitiateUpdate_UpdateError(t *testing.T) {
	mockID := msuuid.UUID(uuid.MustParse("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"))
	repo := &mock.MediaRepo{
		MediaOut:  &model.Media{ID: mockID, Version: 1},
		UpdateErr: errors.New("update failed"),
	}
	strg := &mock.Storage{}
	svc := NewUpdateInitiator(repo, strg)

	_, err := svc.InitiateUpdate(context.Background(), port.InitiateUpdateInput{
		ID:   mockID,
		Name: "file.png",
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if strg.InitiateUploadCalled {
		t.Error("did not expect strg.GeneratePresignedUploadURL to be called")
	}
}

func TestInitiateUpdate_StorageError(t *testing.T) {
	mockID := msuuid.UUID(uuid.MustParse("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"))
	repo := &mock.MediaRepo{
		MediaOut: &model.Media{ID: mockID, Version: 1},
	}
	strg := &mock.Storage{InitiateUploadErr: errors.New("strg failure")}
	svc := NewUpdateInitiator(repo, strg)

	_, err := svc.InitiateUpdate(context.Background(), port.InitiateUpdateInput{
		ID:   mockID,
		Name: "file.png",
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !repo.UpdateCalled {
		t.Error("expected repo.Update to be called")
	}
	if !strg.InitiateUploadCalled {
		t.Error("expected strg.GeneratePresignedUploadURL to be called")
	}
}
