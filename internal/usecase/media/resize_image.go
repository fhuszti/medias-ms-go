package media

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/fhuszti/medias-ms-go/internal/db"
	"github.com/fhuszti/medias-ms-go/internal/model"
	"io"
	"log"
	"path"
	"strings"
)

// ImageResizer resizes images and saves the generated variants.
type ImageResizer interface {
	ResizeImage(ctx context.Context, in ResizeImageInput) error
}

type imageResizerSrv struct {
	repo Repository
	opt  FileOptimiser
	strg Storage
}

// NewImageResizer constructs an ImageResizer implementation.
func NewImageResizer(repo Repository, opt FileOptimiser, strg Storage) ImageResizer {
	return &imageResizerSrv{repo, opt, strg}
}

// ResizeImageInput represents the input for creating resized variants.
type ResizeImageInput struct {
	ID    db.UUID
	Sizes []int
}

// ResizeImage fetches the media by ID and generates resized variants for the given sizes.
func (s *imageResizerSrv) ResizeImage(ctx context.Context, in ResizeImageInput) error {
	media, err := s.repo.GetByID(ctx, in.ID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrObjectNotFound
		}
		return err
	}
	if media.Status != model.MediaStatusCompleted {
		return fmt.Errorf("media status should be 'completed' to be resized")
	}
	if media.MimeType == nil || !IsImage(*media.MimeType) {
		return fmt.Errorf("media is not an image")
	}

	originalReader, err := s.strg.GetFile(ctx, media.Bucket, media.ObjectKey)
	if err != nil {
		return err
	}
	defer func(originalReader io.ReadSeekCloser) { _ = originalReader.Close() }(originalReader)

	for _, width := range in.Sizes {
		if width <= 0 {
			continue
		}
		height := int(float64(media.Metadata.Height) * float64(width) / float64(media.Metadata.Width))

		if _, err := originalReader.Seek(0, io.SeekStart); err != nil {
			return fmt.Errorf("failed to reset reader: %w", err)
		}

		resized, err := s.opt.Resize(*media.MimeType, originalReader, width, height)
		if err != nil {
			return err
		}

		ext := path.Ext(media.ObjectKey)
		base := strings.TrimSuffix(media.ObjectKey, ext)
		variantKey := path.Join(
			"variants",
			media.ID.String(),
			fmt.Sprintf("%s_%d.webp", base, width),
		)
		if err := s.strg.SaveFile(ctx, media.Bucket, variantKey, resized, -1, map[string]string{"Content-Type": "image/webp"}); err != nil {
			_ = resized.Close()
			return fmt.Errorf("failed to save variant %q: %w", variantKey, err)
		}
		_ = resized.Close()

		info, err := s.strg.StatFile(ctx, media.Bucket, variantKey)
		if err != nil {
			return fmt.Errorf("failed reading info about variant %q: %w", variantKey, err)
		}

		media.Variants = append(media.Variants, model.Variant{
			ObjectKey: variantKey,
			SizeBytes: info.SizeBytes,
			Width:     width,
			Height:    height,
		})
	}

	if err := s.repo.Update(ctx, media); err != nil {
		log.Printf("failed updating media with variants: %v", err)
		return fmt.Errorf("failed updating media: %w", err)
	}
	return nil
}
