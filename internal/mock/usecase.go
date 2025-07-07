package mock

import (
	"context"
	"github.com/fhuszti/medias-ms-go/internal/db"
	"github.com/fhuszti/medias-ms-go/internal/port"
)

type MockMediaGetter struct {
	Out    *port.GetMediaOutput
	Id     db.UUID
	Err    error
	Called bool
}

func (m *MockMediaGetter) GetMedia(ctx context.Context, id db.UUID) (*port.GetMediaOutput, error) {
	m.Id = id
	m.Called = true
	return m.Out, m.Err
}

type MockMediaDeleter struct {
	In  port.DeleteMediaInput
	Err error
}

func (m *MockMediaDeleter) DeleteMedia(ctx context.Context, in port.DeleteMediaInput) error {
	m.In = in
	return m.Err
}

type MockUploadLinkGenerator struct {
	Out port.GenerateUploadLinkOutput
	Err error
	In  port.GenerateUploadLinkInput
}

func (m *MockUploadLinkGenerator) GenerateUploadLink(ctx context.Context, in port.GenerateUploadLinkInput) (port.GenerateUploadLinkOutput, error) {
	m.In = in
	return m.Out, m.Err
}

type MockUploadFinaliser struct {
	In  port.FinaliseUploadInput
	Err error
}

func (m *MockUploadFinaliser) FinaliseUpload(ctx context.Context, in port.FinaliseUploadInput) error {
	m.In = in
	return m.Err
}

type MockMediaOptimiser struct {
	In     port.OptimiseMediaInput
	Called bool
	Err    error
}

func (m *MockMediaOptimiser) OptimiseMedia(ctx context.Context, in port.OptimiseMediaInput) error {
	m.Called = true
	m.In = in
	return m.Err
}

type MockImageResizer struct {
	In     port.ResizeImageInput
	Called bool
	Err    error
}

func (m *MockImageResizer) ResizeImage(ctx context.Context, in port.ResizeImageInput) error {
	m.Called = true
	m.In = in
	return m.Err
}
