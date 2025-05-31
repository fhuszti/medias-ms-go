package media

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/fhuszti/medias-ms-go/internal/db"
	"github.com/fhuszti/medias-ms-go/internal/model"
	"log"
	"time"
)

type Getter interface {
	GetMedia(ctx context.Context, in GetMediaInput) (*GetMediaOutput, error)
}

type mediaGetterSrv struct {
	repo          Repository
	cache         Cache
	getTargetStrg StorageGetter
}

func NewMediaGetter(repo Repository, cache Cache, getTargetStrg StorageGetter) Getter {
	return &mediaGetterSrv{repo, cache, getTargetStrg}
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

func (s *mediaGetterSrv) GetMedia(ctx context.Context, in GetMediaInput) (*GetMediaOutput, error) {
	// Try the cache first
	cachedOut, err := s.cache.GetMediaDetails(ctx, in.ID)
	if err == nil && cachedOut != nil {
		return cachedOut, nil
	}

	media, err := s.repo.GetByID(ctx, in.ID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrObjectNotFound
		}
		return nil, err
	}
	if media.Status != model.MediaStatusCompleted {
		return nil, errors.New("media status should be 'completed' to be returned")
	}

	strg, err := s.getTargetStrg(media.Bucket)
	if err != nil {
		return nil, fmt.Errorf("unknown target bucket %q: %w", media.Bucket, err)
	}

	url, err := strg.GeneratePresignedDownloadURL(ctx, media.ObjectKey, DownloadUrlTTL)
	if err != nil {
		return nil, fmt.Errorf("error generating presigned download URL for file %q: %w", media.ObjectKey, err)
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

	// Store in cache for next time
	_ = s.cache.SetMediaDetails(ctx, media.ID, &output)

	return &output, nil
}
