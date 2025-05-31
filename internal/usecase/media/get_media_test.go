package media

import (
	"context"
	"errors"
	"github.com/fhuszti/medias-ms-go/internal/model"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestGetMedia_RepoError(t *testing.T) {
	repo := &mockRepo{getErr: errors.New("db fail")}
	cache := &mockCache{}
	svc := NewMediaGetter(repo, cache, (&mockStorageGetter{}).Get)

	_, err := svc.GetMedia(context.Background(), GetMediaInput{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestGetMedia_WrongStatus(t *testing.T) {
	mrec := &model.Media{Status: model.MediaStatusPending}
	repo := &mockRepo{mediaRecord: mrec}
	cache := &mockCache{}
	svc := NewMediaGetter(repo, cache, (&mockStorageGetter{}).Get)

	_, err := svc.GetMedia(context.Background(), GetMediaInput{})
	want := "media status should be 'completed' to be returned"
	if err == nil || err.Error() != want {
		t.Fatalf("expected %q, got %v", want, err)
	}
}

func TestGetMedia_UnknownBucket(t *testing.T) {
	mrec := &model.Media{Status: model.MediaStatusCompleted, Bucket: "wrong"}
	repo := &mockRepo{mediaRecord: mrec}
	cache := &mockCache{}
	svc := NewMediaGetter(repo, cache, (&mockStorageGetter{err: errors.New("no such bucket")}).Get)

	_, err := svc.GetMedia(context.Background(), GetMediaInput{})
	wantPrefix := "unknown target bucket"
	if err == nil || !strings.HasPrefix(err.Error(), wantPrefix) {
		t.Fatalf("expected prefix %q, got %v", wantPrefix, err)
	}
}

func TestGetMedia_URLGenError(t *testing.T) {
	mt := "image/png"
	mrec := &model.Media{Status: model.MediaStatusCompleted, MimeType: &mt}
	repo := &mockRepo{mediaRecord: mrec}
	cache := &mockCache{}
	strg := &mockStorage{generateDownloadLinkError: errors.New("link generation failed")}
	svc := NewMediaGetter(repo, cache, (&mockStorageGetter{strg: strg}).Get)

	_, err := svc.GetMedia(context.Background(), GetMediaInput{})
	wantPrefix := "error generating presigned download URL"
	if err == nil || !strings.HasPrefix(err.Error(), wantPrefix) {
		t.Fatalf("expected error prefix %q, got %v", wantPrefix, err)
	}
}

func TestGetMedia_CacheSuccess(t *testing.T) {
	cacheOut := &GetMediaOutput{
		ValidUntil: time.Now().Add(1 * time.Hour),
		Optimised:  true,
		URL:        "https://example.com/foo.png",
		Metadata: MetadataOutput{
			Metadata: model.Metadata{
				Width:  1800,
				Height: 1800,
			},
			SizeBytes: int64(1234),
			MimeType:  "image/png",
		},
		Variants: model.VariantsOutput{
			model.VariantOutput{
				URL:       "https://example.com/variants/foo_200.png",
				SizeBytes: 200,
				Width:     200,
				Height:    200,
			},
			model.VariantOutput{
				URL:       "https://example.com/variants/foo_500.png",
				SizeBytes: 500,
				Width:     500,
				Height:    500,
			},
		},
	}
	repo := &mockRepo{}
	cache := &mockCache{out: cacheOut}
	strg := &mockStorage{}
	svc := NewMediaGetter(repo, cache, (&mockStorageGetter{strg: strg}).Get)

	out, err := svc.GetMedia(context.Background(), GetMediaInput{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if repo.getCalled {
		t.Errorf("repo GetById should not be called")
	}
	if strg.generateDownloadLinkCalled {
		t.Errorf("storage GeneratePresignedDownloadURL should not be called")
	}
	if cache.setMediaCalled {
		t.Errorf("cache SetMedia should not be called")
	}

	if !reflect.DeepEqual(out, cacheOut) {
		t.Errorf("Output struct = %+v, want %+v", out, cacheOut)
	}
}

func TestGetMedia_VariantSuccess(t *testing.T) {
	mt := "image/png"
	sb := int64(1234)
	mrec := &model.Media{
		Status:    model.MediaStatusCompleted,
		MimeType:  &mt,
		ObjectKey: "foo.png",
		SizeBytes: &sb,
		Metadata: model.Metadata{
			Width:  1800,
			Height: 1800,
		},
		Variants: model.Variants{
			model.Variant{
				ObjectKey: "variants/foo_200.png",
				SizeBytes: 200,
				Width:     200,
				Height:    200,
			},
			model.Variant{
				ObjectKey: "variants/foo_500.png",
				SizeBytes: 500,
				Width:     500,
				Height:    500,
			},
		},
	}
	repo := &mockRepo{mediaRecord: mrec}
	cache := &mockCache{}
	strg := &mockStorage{}
	svc := NewMediaGetter(repo, cache, (&mockStorageGetter{strg: strg}).Get)

	out, err := svc.GetMedia(context.Background(), GetMediaInput{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	wantKey := "variants/foo_500.png"
	if strg.objectKey != wantKey {
		t.Errorf("Variant key should be %q, got %q", wantKey, strg.objectKey)
	}
	if strg.ttl != DownloadUrlTTL {
		t.Errorf("GeneratePresignedDownloadURL got ttl %v, want %v", strg.ttl, DownloadUrlTTL)
	}

	if !cache.setMediaCalled {
		t.Errorf("cache SetMedia should be called")
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

	if out.Variants[0].URL != "https://example.com/upload" {
		t.Errorf("Variants[0].URL = %q, want 'https://example.com/upload'", out.Variants[0].URL)
	}
	if out.Variants[0].SizeBytes != mrec.Variants[0].SizeBytes {
		t.Errorf("Variants[0].SizeBytes = %d, want %d", out.Variants[0].SizeBytes, mrec.Variants[0].SizeBytes)
	}
	if out.Variants[0].Width != mrec.Variants[0].Width {
		t.Errorf("Variants[0].Width = %d, want %d", out.Variants[0].Width, mrec.Variants[0].Width)
	}
	if out.Variants[0].Height != mrec.Variants[0].Height {
		t.Errorf("Variants[0].Height = %d, want %d", out.Variants[0].Height, mrec.Variants[0].Height)
	}

	if out.Variants[1].URL != "https://example.com/upload" {
		t.Errorf("Variants[1].URL = %q, want 'https://example.com/upload'", out.Variants[1].URL)
	}
	if out.Variants[1].SizeBytes != mrec.Variants[1].SizeBytes {
		t.Errorf("Variants[1].SizeBytes = %d, want %d", out.Variants[1].SizeBytes, mrec.Variants[1].SizeBytes)
	}
	if out.Variants[1].Width != mrec.Variants[1].Width {
		t.Errorf("Variants[1].Width = %d, want %d", out.Variants[1].Width, mrec.Variants[1].Width)
	}
	if out.Variants[1].Height != mrec.Variants[1].Height {
		t.Errorf("Variants[1].Height = %d, want %d", out.Variants[1].Height, mrec.Variants[1].Height)
	}
}
