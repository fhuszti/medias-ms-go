package media

import (
	"context"
	"errors"
	"fmt"
	"github.com/fhuszti/medias-ms-go/internal/db"
	"github.com/fhuszti/medias-ms-go/internal/model"
	"path"
	"strings"
)

type Getter interface {
	GetMedia(ctx context.Context, in GetMediaInput) (GetMediaOutput, error)
}

type mediaGetterSrv struct {
	repo Repository
	strg Storage
}

func NewMediaGetter(repo Repository, strg Storage) Getter {
	return &mediaGetterSrv{repo, strg}
}

type GetMediaInput struct {
	ID    db.UUID
	Width int
}

type GetMediaOutput struct {
	URL      string         `json:"url"`
	Metadata model.Metadata `json:"metadata"`
}

func (s *mediaGetterSrv) GetMedia(ctx context.Context, in GetMediaInput) (GetMediaOutput, error) {
	media, err := s.repo.GetByID(ctx, in.ID)
	if err != nil {
		return GetMediaOutput{}, err
	}
	if media.Status != model.MediaStatusCompleted {
		return GetMediaOutput{}, errors.New("media status should be 'completed' to be returned")
	}

	switch {
	case IsImage(*media.MimeType):
		return s.handleImage(ctx, media, in.Width)
	case isDocument(*media.MimeType):
		return s.handleDocument(ctx, media)
	default:
		return GetMediaOutput{}, fmt.Errorf("unknown mime type for media %q: %s", media.ID, *media.MimeType)
	}
}

func (s *mediaGetterSrv) handleImage(ctx context.Context, media *model.Media, w int) (GetMediaOutput, error) {
	variantKey := media.ObjectKey
	if w > 0 {
		// Add the required width as a suffix to the object key
		dir, file := path.Split(media.ObjectKey)
		ext := path.Ext(file)
		name := strings.TrimSuffix(file, ext)
		variantKey = path.Join(dir, "variants", fmt.Sprintf("%s_%d%s", name, w, ext))
	}

	exists, err := s.strg.FileExists(ctx, variantKey)
	if err != nil {
		return GetMediaOutput{}, fmt.Errorf("error checking if file %q already exists: %w", variantKey, err)
	}

	if !exists {
		//TODO copy original file to variant key
	}

	//TODO generate presigned download link

	return GetMediaOutput{}, nil
}

func (s *mediaGetterSrv) handleDocument(ctx context.Context, media *model.Media) (GetMediaOutput, error) {
	//TODO generate presigned download link

	return GetMediaOutput{}, nil
}
