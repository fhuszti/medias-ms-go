package media

import (
	"context"
	"errors"
	"fmt"
	"github.com/fhuszti/medias-ms-go/internal/db"
	"github.com/fhuszti/medias-ms-go/internal/model"
	"log"
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
	ID db.UUID
}

type MetadataOutput struct {
	model.Metadata
	SizeBytes int64  `json:"size_bytes"`
	MimeType  string `json:"mime_type"`
}

type GetMediaOutput struct {
	ValidUntil time.Time            `json:"valid_until"`
	Optimised  bool                 `json:"optimised"`
	URL        string               `json:"url"`
	Metadata   MetadataOutput       `json:"metadata"`
	Variants   model.VariantsOutput `json:"variants"`
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

	return s.handleFile(ctx, strg, media)
}

func (s *mediaGetterSrv) handleFile(ctx context.Context, strg Storage, media *model.Media) (GetMediaOutput, error) {
	url, err := strg.GeneratePresignedDownloadURL(ctx, media.ObjectKey, DownloadUrlTTL)
	if err != nil {
		return GetMediaOutput{}, fmt.Errorf("error generating presigned download URL for file %q: %w", media.ObjectKey, err)
	}

	mt := MetadataOutput{
		Metadata:  media.Metadata,
		SizeBytes: *media.SizeBytes,
		MimeType:  *media.MimeType,
	}
	output := GetMediaOutput{
		ValidUntil: time.Now().Add(DownloadUrlTTL - 5*time.Minute),
		Optimised:  media.Optimised,
		URL:        url,
		Metadata:   mt,
	}

	if IsImage(*media.MimeType) {
		var variants model.VariantsOutput
		for _, v := range media.Variants {
			vUrl, vErr := strg.GeneratePresignedDownloadURL(ctx, v.ObjectKey, DownloadUrlTTL)
			if vErr != nil {
				log.Printf("error generating presigned download URL for variant %q: %+v", v.ObjectKey, vErr)
				continue
			}
			variants = append(variants, model.VariantOutput{
				URL:       vUrl,
				Width:     v.Width,
				SizeBytes: v.SizeBytes,
				Height:    v.Height,
			})
		}
		output.Variants = variants
	}

	return output, nil
}
