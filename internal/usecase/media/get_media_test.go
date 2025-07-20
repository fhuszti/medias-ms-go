package media

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/fhuszti/medias-ms-go/internal/mock"
	"github.com/fhuszti/medias-ms-go/internal/model"
	msuuid "github.com/fhuszti/medias-ms-go/internal/uuid"
)

func TestGetMedia_RepoError(t *testing.T) {
	repo := &mock.MediaRepo{GetByIDErr: errors.New("db fail")}
	strg := &mock.Storage{}
	svc := NewMediaGetter(repo, strg)

	_, err := svc.GetMedia(context.Background(), msuuid.UUID{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestGetMedia_WrongStatus(t *testing.T) {
	mrec := &model.Media{Status: model.MediaStatusPending}
	repo := &mock.MediaRepo{MediaOut: mrec}
	strg := &mock.Storage{}
	svc := NewMediaGetter(repo, strg)

	_, err := svc.GetMedia(context.Background(), msuuid.UUID{})
	want := "media status should be 'completed' to be returned"
	if err == nil || err.Error() != want {
		t.Fatalf("expected %q, got %v", want, err)
	}
}

func TestGetMedia_URLGenError(t *testing.T) {
	mt := "image/png"
	mrec := &model.Media{Status: model.MediaStatusCompleted, MimeType: &mt}
	repo := &mock.MediaRepo{MediaOut: mrec}
	strg := &mock.Storage{GenerateDownloadLinkErr: errors.New("link generation failed")}
	svc := NewMediaGetter(repo, strg)

	_, err := svc.GetMedia(context.Background(), msuuid.UUID{})
	wantPrefix := "error generating presigned download URL"
	if err == nil || !strings.HasPrefix(err.Error(), wantPrefix) {
		t.Fatalf("expected error prefix %q, got %v", wantPrefix, err)
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
	repo := &mock.MediaRepo{MediaOut: mrec}
	strg := &mock.Storage{}
	svc := NewMediaGetter(repo, strg)

	out, err := svc.GetMedia(context.Background(), msuuid.UUID{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	wantKey := "variants/foo_500.png"
	if strg.ObjectKey != wantKey {
		t.Errorf("Variant key should be %q, got %q", wantKey, strg.ObjectKey)
	}
	if strg.TTL != DownloadUrlTTL {
		t.Errorf("GeneratePresignedDownloadURL got TTL %v, want %v", strg.TTL, DownloadUrlTTL)
	}

	if out.URL != "https://example.com/download" {
		t.Errorf("URL = %q, want 'https://example.com/download'", out.URL)
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

	if out.Variants[0].URL != "https://example.com/download" {
		t.Errorf("Variants[0].URL = %q, want 'https://example.com/download'", out.Variants[0].URL)
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

	if out.Variants[1].URL != "https://example.com/download" {
		t.Errorf("Variants[1].URL = %q, want 'https://example.com/download'", out.Variants[1].URL)
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
