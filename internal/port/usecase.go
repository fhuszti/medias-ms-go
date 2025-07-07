package port

import (
	"context"
	"time"

	"github.com/fhuszti/medias-ms-go/internal/db"
	"github.com/fhuszti/medias-ms-go/internal/model"
)

type UUIDGen func() db.UUID

// MediaGetter retrieves media information from the repository and storage.
type MediaGetter interface {
	GetMedia(ctx context.Context, id db.UUID) (*GetMediaOutput, error)
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

// MediaDeleter deletes a media and its file.
type MediaDeleter interface {
	DeleteMedia(ctx context.Context, id db.UUID) error
}

// UploadLinkGenerator returns a presigned link to upload a file.
type UploadLinkGenerator interface {
	GenerateUploadLink(ctx context.Context, in GenerateUploadLinkInput) (GenerateUploadLinkOutput, error)
}
type GenerateUploadLinkInput struct {
	Name string
}
type GenerateUploadLinkOutput struct {
	ID  db.UUID `json:"id"`
	URL string  `json:"url"`
}

// UploadFinaliser validates the given media in the staging bucket and moves it to the destination bucket.
type UploadFinaliser interface {
	FinaliseUpload(ctx context.Context, in FinaliseUploadInput) error
}
type FinaliseUploadInput struct {
	ID         db.UUID
	DestBucket string
}

// MediaOptimiser reduces the file size with different techniques.
type MediaOptimiser interface {
	OptimiseMedia(ctx context.Context, id db.UUID) error
}

// ImageResizer resizes images and saves the generated variants.
type ImageResizer interface {
	ResizeImage(ctx context.Context, in ResizeImageInput) error
}
type ResizeImageInput struct {
	ID    db.UUID
	Sizes []int
}

// BacklogOptimiser triggers optimisation for stale medias.
type BacklogOptimiser interface {
	OptimiseBacklog(ctx context.Context) error
}
