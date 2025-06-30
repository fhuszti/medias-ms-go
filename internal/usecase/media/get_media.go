package media

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/fhuszti/medias-ms-go/internal/db"
	"github.com/fhuszti/medias-ms-go/internal/model"
	"github.com/fhuszti/medias-ms-go/internal/port"
)

type Getter interface {
	GetMedia(ctx context.Context, in GetMediaInput) (*GetMediaOutput, error)
}

type mediaGetterSrv struct {
	repo port.MediaRepository
	strg port.Storage
}

func NewMediaGetter(repo port.MediaRepository, strg port.Storage) Getter {
	return &mediaGetterSrv{repo: repo, strg: strg}
}

type GetMediaInput struct {
	ID db.UUID
}

func (s *mediaGetterSrv) GetMedia(ctx context.Context, in GetMediaInput) (*GetMediaOutput, error) {
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

	url, err := s.strg.GeneratePresignedDownloadURL(ctx, media.Bucket, media.ObjectKey, DownloadUrlTTL)
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
			vUrl, vErr := s.strg.GeneratePresignedDownloadURL(ctx, media.Bucket, v.ObjectKey, DownloadUrlTTL)
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

	return &output, nil
}
