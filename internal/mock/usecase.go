package mock

import (
	"context"

	"github.com/fhuszti/medias-ms-go/internal/port"
	"github.com/fhuszti/medias-ms-go/internal/uuid"
)

type MediaGetter struct {
	Out    *port.GetMediaOutput
	Id     uuid.UUID
	Err    error
	Called bool
}

func (m *MediaGetter) GetMedia(ctx context.Context, id uuid.UUID) (*port.GetMediaOutput, error) {
	m.Id = id
	m.Called = true
	return m.Out, m.Err
}

type MediaDeleter struct {
	ID  uuid.UUID
	Err error
}

func (m *MediaDeleter) DeleteMedia(ctx context.Context, id uuid.UUID) error {
	m.ID = id
	return m.Err
}

type UploadLinkGenerator struct {
	Out port.GenerateUploadLinkOutput
	Err error
	In  port.GenerateUploadLinkInput
}

func (m *UploadLinkGenerator) GenerateUploadLink(ctx context.Context, in port.GenerateUploadLinkInput) (port.GenerateUploadLinkOutput, error) {
	m.In = in
	return m.Out, m.Err
}

type UploadFinaliser struct {
	In  port.FinaliseUploadInput
	Err error
}

func (m *UploadFinaliser) FinaliseUpload(ctx context.Context, in port.FinaliseUploadInput) error {
	m.In = in
	return m.Err
}

type MediaOptimiser struct {
	ID     uuid.UUID
	Called bool
	Err    error
}

func (m *MediaOptimiser) OptimiseMedia(ctx context.Context, id uuid.UUID) error {
	m.Called = true
	m.ID = id
	return m.Err
}

type ImageResizer struct {
	In     port.ResizeImageInput
	Called bool
	Err    error
}

func (m *ImageResizer) ResizeImage(ctx context.Context, in port.ResizeImageInput) error {
	m.Called = true
	m.In = in
	return m.Err
}
