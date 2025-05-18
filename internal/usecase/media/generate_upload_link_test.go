package media

import (
	"context"
	"errors"
	"io"
	"reflect"
	"testing"
	"time"

	"github.com/fhuszti/medias-ms-go/internal/db"
	"github.com/fhuszti/medias-ms-go/internal/model"
	"github.com/google/uuid"
)

type fakeRepo struct {
	createFn func(ctx context.Context, m *model.Media) error
	mediaArg *model.Media
}

func (f *fakeRepo) Update(ctx context.Context, media *model.Media) error {
	panic("implement me")
}
func (f *fakeRepo) GetByID(ctx context.Context, ID db.UUID) (*model.Media, error) {
	panic("implement me")
}
func (f *fakeRepo) Create(ctx context.Context, m *model.Media) error {
	f.mediaArg = m
	if f.createFn != nil {
		return f.createFn(ctx, m)
	}
	return nil
}

type fakeStorage struct {
	generateFn func(ctx context.Context, objectKey string, ttl time.Duration) (string, error)
	called     bool
	keyArg     string
	ttlArg     time.Duration
}

func (f *fakeStorage) FileExists(ctx context.Context, fileKey string) (bool, error) {
	panic("implement me")
}
func (f *fakeStorage) StatFile(ctx context.Context, fileKey string) (FileInfo, error) {
	panic("implement me")
}
func (f *fakeStorage) RemoveFile(ctx context.Context, fileKey string) error {
	panic("implement me")
}
func (f *fakeStorage) GetFile(ctx context.Context, fileKey string) (io.ReadCloser, error) {
	panic("implement me")
}
func (f *fakeStorage) SaveFile(ctx context.Context, fileKey string, reader io.Reader, fileSize int64, opts map[string]string) error {
	panic("implement me")
}
func (f *fakeStorage) GeneratePresignedUploadURL(ctx context.Context, objectKey string, ttl time.Duration) (string, error) {
	f.called = true
	f.keyArg = objectKey
	f.ttlArg = ttl
	if f.generateFn != nil {
		return f.generateFn(ctx, objectKey, ttl)
	}
	return "", nil
}

func TestGenerateUploadLink_Success(t *testing.T) {
	repo := &fakeRepo{}
	storage := &fakeStorage{
		generateFn: func(ctx context.Context, objectKey string, ttl time.Duration) (string, error) {
			return "https://example.com/upload", nil
		},
	}
	svc := NewUploadLinkGenerator(repo, storage)

	in := GenerateUploadLinkInput{Name: "testName"}
	out, err := svc.GenerateUploadLink(context.Background(), in)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if out.ID == db.UUID(uuid.Nil) {
		t.Error("expected non-zero UUID, got nil")
	}
	if out.URL != "https://example.com/upload" {
		t.Errorf("expected url %q, got %q", "https://example.com/upload", out.URL)
	}

	// verify repo.Create was called with a valid Media
	m := repo.mediaArg
	if m == nil {
		t.Fatal("expected repo.Create to be called")
	}
	if m.ID == db.UUID(uuid.Nil) {
		t.Error("expected non-zero UUID, got nil")
	}
	if m.ObjectKey != m.ID.String() {
		t.Errorf("objectKey %q does not match ID %q", m.ObjectKey, m.ID.String())
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

	// verify storage call
	if !storage.called {
		t.Error("expected storage.GeneratePresignedUploadURL to be called")
	}
	if storage.keyArg != m.ObjectKey {
		t.Errorf("storage called with key %q, want %q", storage.keyArg, m.ObjectKey)
	}
	if storage.ttlArg != 5*time.Minute {
		t.Errorf("storage called with ttl %v, want %v", storage.ttlArg, 5*time.Minute)
	}
}

func TestGenerateUploadLink_RepoError(t *testing.T) {
	repo := &fakeRepo{
		createFn: func(ctx context.Context, m *model.Media) error {
			return errors.New("repo failure")
		},
	}
	storage := &fakeStorage{}
	svc := NewUploadLinkGenerator(repo, storage)

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
	if storage.called {
		t.Error("did not expect storage.GeneratePresignedUploadURL to be called")
	}
}

func TestGenerateUploadLink_StorageError(t *testing.T) {
	repo := &fakeRepo{}
	storage := &fakeStorage{
		generateFn: func(ctx context.Context, objectKey string, ttl time.Duration) (string, error) {
			return "", errors.New("storage failure")
		},
	}
	svc := NewUploadLinkGenerator(repo, storage)

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
	if !storage.called {
		t.Error("expected storage.GeneratePresignedUploadURL to be called")
	}
}
