package media

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/fhuszti/medias-ms-go/internal/db"
	"github.com/fhuszti/medias-ms-go/internal/model"
	"github.com/google/uuid"
)

func TestGenerateUploadLink_Success(t *testing.T) {
	mockID := db.UUID(uuid.MustParse("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"))

	repo := &mockRepo{}
	strg := &mockStorage{}
	svc := NewUploadLinkGenerator(repo, strg, func() db.UUID { return mockID })

	in := GenerateUploadLinkInput{Name: "my-file.webp"}
	out, err := svc.GenerateUploadLink(context.Background(), in)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if out.ID != mockID {
		t.Errorf("expected ID %q, got %q", mockID, out.ID)
	}
	if out.URL != "https://example.com/upload" {
		t.Errorf("expected url %q, got %q", "https://example.com/upload", out.URL)
	}

	// verify repo.Create was called with a valid Media
	m := repo.created
	if m == nil {
		t.Fatal("expected repo.Create to be called")
	}
	if m.ID != mockID {
		t.Errorf("expected create to be called with ID %q, got %q", mockID, out.ID)
	}
	if m.ObjectKey != mockID.String() {
		t.Errorf("objectKey %q does not match ID %q", m.ObjectKey, mockID.String())
	}
	if m.Bucket != "staging" {
		t.Errorf("bucket should be 'staging', got %q", m.Bucket)
	}
	if m.OriginalFilename != in.Name {
		t.Errorf("expected OriginalFilename %q, got %q", in.Name, m.OriginalFilename)
	}
	if m.Status != model.MediaStatusPending {
		t.Errorf("expected Status Pending, got %v", m.Status)
	}
	if !reflect.DeepEqual(m.Metadata, model.Metadata{}) {
		t.Errorf("expected empty Metadata struct, got %+v", m.Metadata)
	}
	if !reflect.DeepEqual(m.Variants, model.Variants{}) {
		t.Errorf("expected empty Variants slice, got %+v", m.Variants)
	}

	// verify strg call
	if !strg.generateUploadLinkCalled {
		t.Error("expected strg.GeneratePresignedUploadURL to be called")
	}
	if strg.objectKey != m.ObjectKey {
		t.Errorf("strg called with key %q, want %q", strg.objectKey, m.ObjectKey)
	}
	if strg.ttl != 5*time.Minute {
		t.Errorf("strg called with ttl %v, want %v", strg.ttl, 5*time.Minute)
	}
}

func TestGenerateUploadLink_RepoError(t *testing.T) {
	mockID := db.UUID(uuid.MustParse("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"))

	repo := &mockRepo{createErr: errors.New("repo failure")}
	strg := &mockStorage{}
	svc := NewUploadLinkGenerator(repo, strg, func() db.UUID { return mockID })

	out, err := svc.GenerateUploadLink(context.Background(), GenerateUploadLinkInput{Name: "foo"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if out.ID != db.UUID(uuid.Nil) {
		t.Errorf("expected zero UUID, got %q", out.ID)
	}
	if out.URL != "" {
		t.Errorf("expected empty url, got %q", out.URL)
	}

	if strg.generateUploadLinkCalled {
		t.Error("did not expect strg.GeneratePresignedUploadURL to be called")
	}
}

func TestGenerateUploadLink_StorageError(t *testing.T) {
	mockID := db.UUID(uuid.MustParse("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"))

	repo := &mockRepo{}
	strg := &mockStorage{generateUploadLinkErr: errors.New("strg failure")}
	svc := NewUploadLinkGenerator(repo, strg, func() db.UUID { return mockID })

	out, err := svc.GenerateUploadLink(context.Background(), GenerateUploadLinkInput{Name: "foo"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if out.ID != db.UUID(uuid.Nil) {
		t.Errorf("expected zero UUID, got %q", out.ID)
	}
	if out.URL != "" {
		t.Errorf("expected empty url, got %q", out.URL)
	}
	if !strg.generateUploadLinkCalled {
		t.Error("expected strg.GeneratePresignedUploadURL to be called")
	}
}
