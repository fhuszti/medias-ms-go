package media

import (
	"context"
	"errors"
	"fmt"
	"github.com/fhuszti/medias-ms-go/internal/db"
	"github.com/fhuszti/medias-ms-go/internal/model"
	"path"
	"strings"
	"time"
)

type Getter interface {
	GetMedia(ctx context.Context, in GetMediaInput) (GetMediaOutput, error)
}

type mediaGetterSrv struct {
	repo          Repository
	getTargetStrg StorageGetter
}

func NewMediaGetter(repo Repository, getTargetStrg StorageGetter) Getter {
	return &mediaGetterSrv{repo, getTargetStrg}
}

type GetMediaInput struct {
	ID    db.UUID
	Width int
}

type MetadataOutput struct {
	model.Metadata
	SizeBytes int64  `json:"size_bytes"`
	MimeType  string `json:"mime_type"`
}

type VariantOutput struct {
	URL       string `json:"url"`
	Width     int    `json:"width"`
	Height    int    `json:"height"`
	SizeBytes int64  `json:"size_bytes"`
}

type GetMediaOutput struct {
	ValidUntil time.Time       `json:"valid_until"`
	URL        string          `json:"url"`
	Metadata   MetadataOutput  `json:"metadata"`
	Variants   []VariantOutput `json:"variants"`
}

func (s *mediaGetterSrv) GetMedia(ctx context.Context, in GetMediaInput) (GetMediaOutput, error) {
	media, err := s.repo.GetByID(ctx, in.ID)
	if err != nil {
		return GetMediaOutput{}, err
	}
	if media.Status != model.MediaStatusCompleted {
		return GetMediaOutput{}, errors.New("media status should be 'completed' to be returned")
	}

	strg, err := s.getTargetStrg(media.Bucket)
	if err != nil {
		return GetMediaOutput{}, fmt.Errorf("unknown target bucket %q: %w", media.Bucket, err)
	}

	switch {
	case IsImage(*media.MimeType):
		return s.handleImage(ctx, strg, media, in.Width)
	case isDocument(*media.MimeType):
		return s.handleDocument(ctx, strg, media)
	default:
		return GetMediaOutput{}, fmt.Errorf("unknown mime type for media %q: %s", media.ID, *media.MimeType)
	}
}

func (s *mediaGetterSrv) handleImage(ctx context.Context, strg Storage, media *model.Media, w int) (GetMediaOutput, error) {
	variantKey := media.ObjectKey
	if w > 0 {
		// Add the required width as a suffix to the object key
		dir, file := path.Split(media.ObjectKey)
		ext := path.Ext(file)
		name := strings.TrimSuffix(file, ext)
		variantKey = path.Join(dir, "variants", fmt.Sprintf("%s_%d%s", name, w, ext))
	}

	exists, err := strg.FileExists(ctx, variantKey)
	if err != nil {
		return GetMediaOutput{}, fmt.Errorf("error checking if file %q already exists: %w", variantKey, err)
	}

	if !exists {
		if err := strg.CopyFile(ctx, media.ObjectKey, variantKey); err != nil {
			return GetMediaOutput{}, fmt.Errorf("error copying placeholder variant image: %w", err)
		}
	}

	url, err := strg.GeneratePresignedDownloadURL(ctx, variantKey, DownloadUrlTTL)
	if err != nil {
		return GetMediaOutput{}, fmt.Errorf("error generating presigned download URL for file %q: %w", variantKey, err)
	}

	mt := MetadataOutput{
		Metadata:  media.Metadata,
		SizeBytes: *media.SizeBytes,
		MimeType:  *media.MimeType,
	}

	return GetMediaOutput{
		ValidUntil: time.Now().Add(DownloadUrlTTL - 5*time.Minute),
		URL:        url,
		Metadata:   mt,
	}, nil
}

func (s *mediaGetterSrv) handleDocument(ctx context.Context, strg Storage, media *model.Media) (GetMediaOutput, error) {
	url, err := strg.GeneratePresignedDownloadURL(ctx, media.ObjectKey, DownloadUrlTTL)
	if err != nil {
		return GetMediaOutput{}, fmt.Errorf("error generating presigned download URL for file %q: %w", media.ObjectKey, err)
	}

	mt := MetadataOutput{
		Metadata:  media.Metadata,
		SizeBytes: *media.SizeBytes,
		MimeType:  *media.MimeType,
	}

	return GetMediaOutput{
		ValidUntil: time.Now().Add(DownloadUrlTTL - 5*time.Minute),
		URL:        url,
		Metadata:   mt,
	}, nil
}
