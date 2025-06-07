package media

import (
	"database/sql"
	"errors"
	"fmt"
	"github.com/fhuszti/medias-ms-go/internal/db"
	"github.com/fhuszti/medias-ms-go/internal/model"
	"golang.org/x/net/context"
	"io"
	"log"
	"path/filepath"
	"strings"
)

type Optimiser interface {
	OptimiseMedia(ctx context.Context, in OptimiseMediaInput) error
}

type mediaOptimiserSrv struct {
	repo Repository
	opt  FileOptimiser
	strg Storage
}

func NewMediaOptimiser(repo Repository, opt FileOptimiser, strg Storage) Optimiser {
	return &mediaOptimiserSrv{repo, opt, strg}
}

type OptimiseMediaInput struct {
	ID db.UUID
}

func (m *mediaOptimiserSrv) OptimiseMedia(ctx context.Context, in OptimiseMediaInput) error {
	media, err := m.repo.GetByID(ctx, in.ID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrObjectNotFound
		}
		return err
	}
	if media.Status != model.MediaStatusCompleted {
		return errors.New("media status should be 'completed' to be optimised")
	}

	originalReader, err := m.strg.GetFile(ctx, media.Bucket, media.ObjectKey)
	if err != nil {
		return err
	}
	defer func(originalReader io.ReadSeekCloser) {
		_ = originalReader.Close()
	}(originalReader)

	// Actually do the compression here
	compressedReader, newMimeType, err := m.opt.Compress(*media.MimeType, originalReader)
	if err != nil {
		return err
	}
	defer func(compressedReader io.ReadCloser) {
		_ = compressedReader.Close()
	}(compressedReader)

	newObjectKey := media.ObjectKey
	if newMimeType != *media.MimeType {
		ext, err := MimeTypeToExtension(newMimeType)
		if err != nil {
			return err
		}
		// Update extension in object key
		newObjectKey = strings.TrimSuffix(media.ObjectKey, filepath.Ext(media.ObjectKey)) + ext
	}

	// Save the compressed file to tmp file (failsafe in case it breaks in the middle)
	tempKey := newObjectKey + ".tmp"
	if err := m.strg.SaveFile(
		ctx,
		media.Bucket,
		tempKey,
		compressedReader,
		-1, // streaming mode
		map[string]string{
			"Content-Type": newMimeType,
		},
	); err != nil {
		return fmt.Errorf("failed to save temp file %q inside bucket %q: %w", tempKey, media.Bucket, err)
	}

	// Copy the finished tmp file to its final object key
	if err := m.strg.CopyFile(ctx, media.Bucket, tempKey, newObjectKey); err != nil {
		return fmt.Errorf("failed to copy %q→%q inside bucket %q: %w", tempKey, newObjectKey, media.Bucket, err)
	}

	// Remove the tmp file
	if err := m.strg.RemoveFile(ctx, media.Bucket, tempKey); err != nil {
		log.Printf("warning: failed to remove temp file %q from bucket %q: %v", tempKey, media.Bucket, err)
	}

	// If the file extension has changed, remove the original
	if newObjectKey != media.ObjectKey {
		if err := m.strg.RemoveFile(ctx, media.Bucket, media.ObjectKey); err != nil {
			log.Printf("warning: failed to remove old file %q from bucket %q: %v", media.ObjectKey, media.Bucket, err)
		}
	}

	info, err := m.strg.StatFile(ctx, media.Bucket, newObjectKey)
	if err != nil {
		return fmt.Errorf("failed reading info about file %q inside bucket %q: %w", newObjectKey, media.Bucket, err)
	}
	newSize := info.SizeBytes

	media.Optimised = true
	media.SizeBytes = &newSize
	media.MimeType = &newMimeType
	media.ObjectKey = newObjectKey

	if err := m.repo.Update(ctx, media); err != nil {
		return fmt.Errorf("failed updating media: %w", err)
	}

	return nil
}
