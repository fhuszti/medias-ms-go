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

type UploadInitiator struct {
	In  port.InitiateUploadInput
	Out port.InitiateUploadOutput
	Err error
}

func (m *UploadInitiator) InitiateUpload(ctx context.Context, in port.InitiateUploadInput) (port.InitiateUploadOutput, error) {
	m.In = in
	return m.Out, m.Err
}

type UpdateInitiator struct {
	In  port.InitiateUpdateInput
	Out port.InitiateUpdateOutput
	Err error
}

func (m *UpdateInitiator) InitiateUpdate(ctx context.Context, in port.InitiateUpdateInput) (port.InitiateUpdateOutput, error) {
	m.In = in
	return m.Out, m.Err
}

type UpdateFinaliser struct {
	In  port.FinaliseUpdateInput
	Err error
}

func (m *UpdateFinaliser) FinaliseUpdate(ctx context.Context, in port.FinaliseUpdateInput) error {
	m.In = in
	return m.Err
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
