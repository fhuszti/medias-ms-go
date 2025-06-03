package media

import (
	"bytes"
	"database/sql"
	"errors"
	"fmt"
	"github.com/fhuszti/medias-ms-go/internal/db"
	"github.com/fhuszti/medias-ms-go/internal/model"
	"golang.org/x/net/context"
	"path/filepath"
	"strings"
)

type Optimiser interface {
	OptimiseMedia(ctx context.Context, in OptimiseMediaInput) error
}

type mediaOptimiserSrv struct {
	repo          Repository
	opt           FileOptimiser
	getTargetStrg StorageGetter
}

func NewMediaOptimiser(repo Repository, opt FileOptimiser, getTargetStrg StorageGetter) Optimiser {
	return &mediaOptimiserSrv{repo, opt, getTargetStrg}
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

	strg, err := m.getTargetStrg(media.Bucket)
	if err != nil {
		return err
	}

	file, err := strg.GetFile(ctx, media.ObjectKey)
	if err != nil {
		return err
	}

	compressedFile, mimeType, err := m.opt.Compress(*media.MimeType, file)
	if err != nil {
		return err
	}

	newObjectKey := media.ObjectKey
	if mimeType != *media.MimeType {
		ext, err := MimeTypeToExtension(mimeType)
		if err != nil {
			return err
		}
		// Update extension in object key
		newObjectKey = strings.TrimSuffix(media.ObjectKey, filepath.Ext(media.ObjectKey)) + ext
	}

	newSize := int64(len(compressedFile))

	if err := strg.SaveFile(
		ctx,
		newObjectKey,
		bytes.NewReader(compressedFile),
		newSize,
		map[string]string{
			"Content-Type": mimeType,
		},
	); err != nil {
		return err
	}

	media.Optimised = true
	media.SizeBytes = &newSize
	media.MimeType = &mimeType
	media.ObjectKey = newObjectKey

	if err := m.repo.Update(ctx, media); err != nil {
		return fmt.Errorf("failed updating media: %w", err)
	}

	return nil
}
