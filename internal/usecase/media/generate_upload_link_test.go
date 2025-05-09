// internal/media/service_test.go
package media

import (
	"context"
	"errors"
	"github.com/fhuszti/medias-ms-go/internal/db"
	"regexp"
	"testing"
	"time"

	"github.com/fhuszti/medias-ms-go/internal/model"
	"github.com/google/uuid"
)

type fakeRepo struct {
	createFn func(ctx context.Context, m *model.Media) error
	mediaArg *model.Media
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

func (f *fakeStorage) GeneratePresignedDownloadURL(ctx context.Context, objectKey string, expiry time.Duration, downloadName string, inline bool) (string, error) {
	panic("implement me")
}

func (f *fakeStorage) ObjectExists(ctx context.Context, objectKey string) (bool, error) {
	panic("implement me")
}

func (f *fakeStorage) PublicURL(objectKey string) string {
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

	in := GenerateUploadLinkInput{Name: "testName", Type: "image/png"}
	gotURL, err := svc.GenerateUploadLink(context.Background(), in)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if gotURL != "https://example.com/upload" {
		t.Errorf("expected url %q, got %q", "https://example.com/upload", gotURL)
	}

	// verify repo.Create was called with a valid Media
	m := repo.mediaArg
	if m == nil {
		t.Fatal("expected repo.Create to be called")
	}
	// ID should not be the nil-UUID
	if m.ID == db.UUID(uuid.Nil) {
		t.Error("expected non-zero UUID, got nil")
	}
	// objectKey should be "<Name>_<unixNano>"
	pattern := `^testName_\d+$`
	if matched, _ := regexp.MatchString(pattern, m.ObjectKey); !matched {
		t.Errorf("objectKey %q does not match %q", m.ObjectKey, pattern)
	}
	// MimeType & Status
	if m.MimeType != in.Type {
		t.Errorf("expected MimeType %q, got %q", in.Type, m.MimeType)
	}
	if m.Status != model.MediaStatusPending {
		t.Errorf("expected Status Pending, got %v", m.Status)
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

	url, err := svc.GenerateUploadLink(context.Background(), GenerateUploadLinkInput{Name: "foo", Type: "bar"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if url != "" {
		t.Errorf("expected empty url, got %q", url)
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

	url, err := svc.GenerateUploadLink(context.Background(), GenerateUploadLinkInput{Name: "foo", Type: "bar"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if url != "" {
		t.Errorf("expected empty url, got %q", url)
	}
	if !storage.called {
		t.Error("expected storage.GeneratePresignedUploadURL to be called")
	}
}
