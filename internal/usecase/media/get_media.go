package media

import (
	"context"
	"errors"
	"fmt"
	"github.com/fhuszti/medias-ms-go/internal/db"
	"github.com/fhuszti/medias-ms-go/internal/model"
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
	URL string `json:"url"`
}

func (s *mediaGetterSrv) GetMedia(ctx context.Context, in GetMediaInput) (GetMediaOutput, error) {
	media, err := s.repo.GetByID(ctx, in.ID)
	if err != nil {
		return GetMediaOutput{}, err
	}
	if media.Status != model.MediaStatusCompleted {
		return GetMediaOutput{}, errors.New("media status should be 'completed' to be finalised")
	}

	switch {
	case IsImage(*media.MimeType):
		return handleImage(media)
	case IsPdf(*media.MimeType):
		return handlePdf(media)
	case IsMarkdown(*media.MimeType):
		return handleMarkdown(media)
	default:
		return GetMediaOutput{}, fmt.Errorf("unknown mime type for media %q: %s", media.ID, *media.MimeType)
	}
}

func handleImage(media *model.Media) (GetMediaOutput, error) {
	return GetMediaOutput{}, nil
}

func handlePdf(media *model.Media) (GetMediaOutput, error) {
	return GetMediaOutput{}, nil
}

func handleMarkdown(media *model.Media) (GetMediaOutput, error) {
	return GetMediaOutput{}, nil
}
