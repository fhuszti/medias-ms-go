package media

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	_ "golang.org/x/image/webp"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"log"
	"regexp"
	"strings"

	"github.com/fhuszti/medias-ms-go/internal/model"
	"github.com/fhuszti/medias-ms-go/internal/port"
	"github.com/ledongthuc/pdf"
)

type uploadFinaliserSrv struct {
	repo  port.MediaRepository
	strg  port.Storage
	tasks port.TaskDispatcher
}

// compile-time check: *uploadFinaliserSrv must satisfy port.UploadFinaliser
var _ port.UploadFinaliser = (*uploadFinaliserSrv)(nil)

func NewUploadFinaliser(repo port.MediaRepository, strg port.Storage, tasks port.TaskDispatcher) port.UploadFinaliser {
	return &uploadFinaliserSrv{repo, strg, tasks}
}

func (s *uploadFinaliserSrv) FinaliseUpload(ctx context.Context, in port.FinaliseUploadInput) error {
	media, err := s.repo.GetByID(ctx, in.ID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrObjectNotFound
		}
		return err
	}
	if media.Status == model.MediaStatusCompleted {
		return nil
	}
	if media.Status != model.MediaStatusPending {
		return errors.New("media status should be 'pending' to be finalised")
	}

	// Cleanup function
	var finalErr error
	defer func() {
		if finalErr != nil {
			if err := s.cleanupFile(media.ObjectKey); err != nil {
				log.Printf("cleanup failed for file %q: %v", media.ObjectKey, err)
			}
			if markErr := s.markAsFailed(ctx, media, finalErr.Error()); markErr != nil {
				log.Printf("markAsFailed failed for file %q: %v", media.ObjectKey, markErr)
			}
		}
	}()

	info, err := s.strg.StatFile(ctx, "staging", media.ObjectKey)
	if err != nil {
		if errors.Is(err, ErrObjectNotFound) {
			finalErr = fmt.Errorf("staging file %q not found", media.ObjectKey)
		} else {
			finalErr = fmt.Errorf("stats for file %q failed: %w", media.ObjectKey, err)
		}
		return finalErr
	}

	if info.SizeBytes < MinFileSize {
		finalErr = fmt.Errorf("file %q too small: %d bytes (min size: %d bytes)", media.ObjectKey, info.SizeBytes, MinFileSize)
		return finalErr
	}
	if info.SizeBytes > MaxFileSize {
		finalErr = fmt.Errorf("file %q too large: %d bytes (max size: %d bytes)", media.ObjectKey, info.SizeBytes, MaxFileSize)
		return finalErr
	}

	if !IsMimeTypeAllowed(info.ContentType) {
		finalErr = fmt.Errorf("unsupported mime-type %q for file %q", info.ContentType, media.ObjectKey)
		return finalErr
	}

	if err := s.moveFile(ctx, media, info.SizeBytes, info.ContentType, in.DestBucket); err != nil {
		finalErr = fmt.Errorf("move file %q from staging to bucket %q failed: %w", media.ObjectKey, in.DestBucket, err)
		return finalErr
	}

	if err := s.tasks.EnqueueOptimiseMedia(ctx, media.ID); err != nil {
		log.Printf("failed to enqueue optimise task for media #%s: %v", media.ID, err)
	}

	return nil
}

func (s *uploadFinaliserSrv) cleanupFile(objectKey string) error {
	if err := s.strg.RemoveFile(context.Background(), "staging", objectKey); err != nil {
		return err
	}
	return nil
}

func (s *uploadFinaliserSrv) markAsFailed(ctx context.Context, media *model.Media, reason string) error {
	media.Status = model.MediaStatusFailed
	media.FailureMessage = &reason

	if err := s.repo.Update(ctx, media); err != nil {
		return err
	}
	return nil
}

func (s *uploadFinaliserSrv) moveFile(ctx context.Context, media *model.Media, size int64, contentType string, destBucket string) error {
	file, err := s.strg.GetFile(ctx, "staging", media.ObjectKey)
	if err != nil {
		return err
	}

	// Read metadata then reset reader position for saving.
	metadata, err := fillMetadata(contentType, file)
	if err != nil {
		return fmt.Errorf("failed to fill metadata: %w", err)
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return fmt.Errorf("failed to reset reader: %w", err)
	}
	defer func(file io.ReadSeekCloser) {
		if err := file.Close(); err != nil {
			log.Printf("failed to close reader")
		}
	}(file)

	ext, err := MimeTypeToExtension(contentType)
	if err != nil {
		return err
	}
	newObjectKey := fmt.Sprintf("%s%s", media.ObjectKey, ext)

	if err := s.strg.SaveFile(
		ctx,
		destBucket,
		newObjectKey,
		file,
		size,
		map[string]string{
			"Content-Type": contentType,
		},
	); err != nil {
		return err
	}

	if err := s.strg.RemoveFile(ctx, "staging", media.ObjectKey); err != nil {
		log.Printf("failed to clean up file %q in staging: %v", media.ObjectKey, err)
	}

	updated := *media
	updated.ObjectKey = newObjectKey
	updated.Bucket = destBucket
	updated.Status = model.MediaStatusCompleted
	updated.SizeBytes = &size
	updated.MimeType = &contentType
	updated.Metadata = metadata

	if err := s.repo.Update(ctx, &updated); err != nil {
		if remErr := s.strg.RemoveFile(ctx, destBucket, newObjectKey); remErr != nil {
			log.Printf("failed to remove file %q from bucket %q after update failure: %v", newObjectKey, destBucket, remErr)
		}
		return fmt.Errorf("failed updating media: %w", err)
	}

	*media = updated

	return nil
}

func fillMetadata(mimeType string, file io.Reader) (model.Metadata, error) {

	switch {
	case IsImage(mimeType):
		return fillImageMetadata(file)
	case IsPdf(mimeType):
		return fillPdfMetadata(file)
	case IsMarkdown(mimeType):
		return fillMarkdownMetadata(file)
	default:
		return model.Metadata{}, errors.New("unsupported mime-type")
	}
}

func fillImageMetadata(file io.Reader) (model.Metadata, error) {
	data, err := io.ReadAll(file)
	if err != nil {
		return model.Metadata{}, fmt.Errorf("error reading image data: %w", err)
	}

	cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return model.Metadata{}, fmt.Errorf("error decoding image config: %w", err)
	}

	return model.Metadata{
		Width:  cfg.Width,
		Height: cfg.Height,
	}, nil
}

func fillPdfMetadata(file io.Reader) (model.Metadata, error) {
	data, err := io.ReadAll(file)
	if err != nil {
		return model.Metadata{}, fmt.Errorf("error reading PDF data: %w", err)
	}

	reader, err := pdf.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return model.Metadata{}, fmt.Errorf("error opening pdf reader: %w", err)
	}

	return model.Metadata{
		PageCount: reader.NumPage(),
	}, nil
}

func fillMarkdownMetadata(file io.Reader) (model.Metadata, error) {
	data, err := io.ReadAll(file)
	if err != nil {
		return model.Metadata{}, fmt.Errorf("error reading markdown data: %w", err)
	}
	text := string(data)

	words := strings.Fields(text)

	headingCount := 0
	for _, line := range strings.Split(text, "\n") {
		if strings.HasPrefix(line, "# ") || strings.HasPrefix(line, "##") {
			headingCount++
		}
	}

	linkRe := regexp.MustCompile(`\[[^]]+]\([^)]+\)`)
	links := linkRe.FindAllString(text, -1)

	return model.Metadata{
		WordCount:    int64(len(words)),
		HeadingCount: int64(headingCount),
		LinkCount:    int64(len(links)),
	}, nil
}
