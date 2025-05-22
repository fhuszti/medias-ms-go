package media

import (
	"context"
	"errors"
	"github.com/fhuszti/medias-ms-go/internal/model"
	"reflect"
	"strings"
	"testing"
)

func TestGetMedia_RepoError(t *testing.T) {
	repo := &mockRepo{getErr: errors.New("db fail")}
	svc := NewMediaGetter(repo, (&mockStorageGetter{}).Get)

	_, err := svc.GetMedia(context.Background(), GetMediaInput{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestGetMedia_WrongStatus(t *testing.T) {
	mrec := &model.Media{Status: model.MediaStatusPending}
	repo := &mockRepo{mediaRecord: mrec}
	svc := NewMediaGetter(repo, (&mockStorageGetter{}).Get)

	_, err := svc.GetMedia(context.Background(), GetMediaInput{})
	want := "media status should be 'completed' to be returned"
	if err == nil || err.Error() != want {
		t.Fatalf("expected %q, got %v", want, err)
	}
}

func TestGetMedia_UnknownBucket(t *testing.T) {
	mrec := &model.Media{Status: model.MediaStatusCompleted, Bucket: "wrong"}
	repo := &mockRepo{mediaRecord: mrec}
	svc := NewMediaGetter(repo, (&mockStorageGetter{err: errors.New("no such bucket")}).Get)

	_, err := svc.GetMedia(context.Background(), GetMediaInput{})
	wantPrefix := "unknown target bucket"
	if err == nil || !strings.HasPrefix(err.Error(), wantPrefix) {
		t.Fatalf("expected prefix %q, got %v", wantPrefix, err)
	}
}

func TestGetMedia_UnknownMimeType(t *testing.T) {
	mt := "wrong/type"
	mrec := &model.Media{Status: model.MediaStatusCompleted, MimeType: &mt}
	repo := &mockRepo{mediaRecord: mrec}
	svc := NewMediaGetter(repo, (&mockStorageGetter{}).Get)

	_, err := svc.GetMedia(context.Background(), GetMediaInput{})
	wantPrefix := "unknown mime type"
	if err == nil || !strings.HasPrefix(err.Error(), wantPrefix) {
		t.Fatalf("expected %q, got %v", wantPrefix, err)
	}
}

func TestGetMedia_HandleImage_FileExistsError(t *testing.T) {
	mt := "image/png"
	mrec := &model.Media{Status: model.MediaStatusCompleted, MimeType: &mt}
	repo := &mockRepo{mediaRecord: mrec}
	strg := &mockStorage{fileExistsErr: errors.New("err on file exists")}
	svc := NewMediaGetter(repo, (&mockStorageGetter{strg: strg}).Get)

	_, err := svc.GetMedia(context.Background(), GetMediaInput{})
	if err == nil || !strings.HasSuffix(err.Error(), "err on file exists") {
		t.Fatalf("expected file exists error, got %v", err)
	}
}

func TestGetMedia_HandleImage_CopyError(t *testing.T) {
	mt := "image/png"
	mrec := &model.Media{Status: model.MediaStatusCompleted, MimeType: &mt}
	repo := &mockRepo{mediaRecord: mrec}
	strg := &mockStorage{copyErr: errors.New("disk full")}
	svc := NewMediaGetter(repo, (&mockStorageGetter{strg: strg}).Get)

	_, err := svc.GetMedia(context.Background(), GetMediaInput{})
	wantPrefix := "error copying placeholder"
	if err == nil || !strings.HasPrefix(err.Error(), wantPrefix) {
		t.Fatalf("expected error prefix %q, got %v", wantPrefix, err)
	}
}

func TestGetMedia_HandleImage_URLGenError(t *testing.T) {
	mt := "image/png"
	mrec := &model.Media{Status: model.MediaStatusCompleted, MimeType: &mt}
	repo := &mockRepo{mediaRecord: mrec}
	strg := &mockStorage{generateDownloadLinkError: errors.New("link generation failed")}
	svc := NewMediaGetter(repo, (&mockStorageGetter{strg: strg}).Get)

	_, err := svc.GetMedia(context.Background(), GetMediaInput{})
	wantPrefix := "error generating presigned download URL"
	if err == nil || !strings.HasPrefix(err.Error(), wantPrefix) {
		t.Fatalf("expected error prefix %q, got %v", wantPrefix, err)
	}
}

func TestGetMedia_HandleImage_VariantExists(t *testing.T) {
	mt := "image/png"
	sb := int64(1234)
	mrec := &model.Media{Status: model.MediaStatusCompleted, MimeType: &mt, ObjectKey: "foo.png", SizeBytes: &sb}
	repo := &mockRepo{mediaRecord: mrec}
	strg := &mockStorage{fileExists: true}
	svc := NewMediaGetter(repo, (&mockStorageGetter{strg: strg}).Get)

	in := GetMediaInput{Width: 200}
	out, err := svc.GetMedia(context.Background(), in)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	wantKey := "variants/foo_200.png"

	if strg.copyCalled {
		t.Error("storage copy was called when it should not")
	}
	if strg.objectKey != wantKey {
		t.Errorf("Variant key should be %q, got %q", wantKey, strg.objectKey)
	}
	if strg.ttl != DownloadUrlTTL {
		t.Errorf("GeneratePresignedDownloadURL got ttl %v, want %v", strg.ttl, DownloadUrlTTL)
	}

	if out.URL != "https://example.com/upload" {
		t.Errorf("URL = %q, want 'https://example.com/upload'", out.URL)
	}
	if out.Metadata.MimeType != *mrec.MimeType {
		t.Errorf("MimeType = %q, want %q", out.Metadata.MimeType, *mrec.MimeType)
	}
	if out.Metadata.SizeBytes != *mrec.SizeBytes {
		t.Errorf("SizeBytes = %d, want %d", out.Metadata.SizeBytes, *mrec.SizeBytes)
	}
	if !reflect.DeepEqual(out.Metadata.Metadata, mrec.Metadata) {
		t.Errorf("Metadata struct = %+v, want %+v", out.Metadata.Metadata, mrec.Metadata)
	}
}
