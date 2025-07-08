package media

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"log"
	"path"
	"strings"

	"github.com/fhuszti/medias-ms-go/internal/model"
	"github.com/fhuszti/medias-ms-go/internal/port"
)

type imageResizerSrv struct {
	repo  port.MediaRepository
	opt   port.FileOptimiser
	strg  port.Storage
	cache port.Cache
}

// compile-time check: *imageResizerSrv must satisfy port.ImageResizer
var _ port.ImageResizer = (*imageResizerSrv)(nil)

// NewImageResizer constructs an ImageResizer implementation.
func NewImageResizer(repo port.MediaRepository, opt port.FileOptimiser, strg port.Storage, cache port.Cache) port.ImageResizer {
	return &imageResizerSrv{repo, opt, strg, cache}
}

// ResizeImage fetches the media by ID and generates resized variants for the given sizes.
func (s *imageResizerSrv) ResizeImage(ctx context.Context, in port.ResizeImageInput) error {
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

		ext := path.Ext(media.ObjectKey)
		base := strings.TrimSuffix(media.ObjectKey, ext)
		variantKey := path.Join(
			"variants",
			media.ID.String(),
			fmt.Sprintf("%s_%d.webp", base, width),
		)

		var (
			variantWidth  int
			variantHeight int
		)

		if width < media.Metadata.Width {
			variantWidth = width
			variantHeight = int(float64(media.Metadata.Height) * float64(width) / float64(media.Metadata.Width))

			if _, err := originalReader.Seek(0, io.SeekStart); err != nil {
				return fmt.Errorf("failed to reset reader: %w", err)
			}

			resized, err := s.opt.Resize(*media.MimeType, originalReader, variantWidth, variantHeight)
			if err != nil {
				return err
			}

			if err := s.strg.SaveFile(ctx, media.Bucket, variantKey, resized, -1, map[string]string{"Content-Type": "image/webp"}); err != nil {
				_ = resized.Close()
				return fmt.Errorf("failed to save variant %q: %w", variantKey, err)
			}
			_ = resized.Close()
		} else {
			variantWidth = media.Metadata.Width
			variantHeight = media.Metadata.Height

			if err := s.strg.CopyFile(ctx, media.Bucket, media.ObjectKey, variantKey); err != nil {
				return fmt.Errorf("failed to copy original file to variant %q: %w", variantKey, err)
			}
		}
		info, err := s.strg.StatFile(ctx, media.Bucket, variantKey)
		if err != nil {
			return fmt.Errorf("failed reading info about variant %q: %w", variantKey, err)
		}

		media.Variants = append(media.Variants, model.Variant{
			ObjectKey: variantKey,
			SizeBytes: info.SizeBytes,
			Width:     variantWidth,
			Height:    variantHeight,
		})
	}

	if err := s.repo.Update(ctx, media); err != nil {
		log.Printf("failed updating media with variants: %v", err)
		return fmt.Errorf("failed updating media: %w", err)
	}

	if err := s.cache.DeleteMediaDetails(ctx, media.ID); err != nil {
		log.Printf("failed deleting cache for media #%s: %v", media.ID, err)
	}
	if err := s.cache.DeleteEtagMediaDetails(ctx, media.ID); err != nil {
		log.Printf("failed deleting etag cache for media #%s: %v", media.ID, err)
	}
	return nil
}
