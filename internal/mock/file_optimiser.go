package mock

import (
	"bytes"
	"io"
)

// FileOptimiser implements file optimisation operations for tests.
type FileOptimiser struct {
	// stored values
	CompressOut []byte
	MimeOut     string
	ResizeOut   []byte

	// errors
	CompressErr error
	ResizeErr   error

	// call flags
	CompressCalled bool
	ResizeCalled   bool
}

func (m *FileOptimiser) Compress(mimeType string, r io.Reader) (io.ReadCloser, string, error) {
	m.CompressCalled = true
	if m.CompressErr != nil {
		return nil, "", m.CompressErr
	}
	return io.NopCloser(bytes.NewReader(m.CompressOut)), m.MimeOut, nil
}

func (m *FileOptimiser) Resize(mimeType string, r io.Reader, width, height int) (io.ReadCloser, error) {
	m.ResizeCalled = true
	if m.ResizeErr != nil {
		return nil, m.ResizeErr
	}
	return io.NopCloser(bytes.NewReader(m.ResizeOut)), nil
}
