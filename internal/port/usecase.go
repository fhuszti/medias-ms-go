package port

import (
	"context"
	"time"

	"github.com/fhuszti/medias-ms-go/internal/model"
	"github.com/fhuszti/medias-ms-go/internal/uuid"
)

type UUIDGen func() uuid.UUID

// MediaGetter retrieves media information from the repository and storage.
type MediaGetter interface {
	GetMedia(ctx context.Context, id uuid.UUID) (*GetMediaOutput, error)
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
	DeleteMedia(ctx context.Context, id uuid.UUID) error
}

// UploadInitiator creates the media record and returns a presigned upload link.
type UploadInitiator interface {
	InitiateUpload(ctx context.Context, in InitiateUploadInput) (InitiateUploadOutput, error)
}
type InitiateUploadInput struct {
	Name string
}
type InitiateUploadOutput struct {
	ID  uuid.UUID `json:"id"`
	URL string    `json:"url"`
}

// UpdateInitiator creates a new staging upload for an existing media.
type UpdateInitiator interface {
	InitiateUpdate(ctx context.Context, in InitiateUpdateInput) (InitiateUpdateOutput, error)
}
type InitiateUpdateInput struct {
	ID   uuid.UUID
	Name string
}
type InitiateUpdateOutput struct {
	ID  uuid.UUID `json:"id"`
	URL string    `json:"url"`
}

// UpdateFinaliser validates the given media update in the staging bucket and moves it to the destination bucket.
type UpdateFinaliser interface {
	FinaliseUpdate(ctx context.Context, in FinaliseUpdateInput) error
}
type FinaliseUpdateInput struct {
	ID         uuid.UUID
	DestBucket string
}

// UploadFinaliser validates the given media in the staging bucket and moves it to the destination bucket.
type UploadFinaliser interface {
	FinaliseUpload(ctx context.Context, in FinaliseUploadInput) error
}
type FinaliseUploadInput struct {
	ID         uuid.UUID
	DestBucket string
}

// MediaOptimiser reduces the file size with different techniques.
type MediaOptimiser interface {
	OptimiseMedia(ctx context.Context, id uuid.UUID) error
}

// ImageResizer resizes images and saves the generated variants.
type ImageResizer interface {
	ResizeImage(ctx context.Context, in ResizeImageInput) error
}
type ResizeImageInput struct {
	ID    uuid.UUID
	Sizes []int
}

// BacklogOptimiser triggers optimisation for stale medias.
type BacklogOptimiser interface {
	OptimiseBacklog(ctx context.Context) error
}
