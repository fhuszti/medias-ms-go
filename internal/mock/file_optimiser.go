package mock

import (
	"bytes"
	"io"
)

// MockFileOptimiser implements file optimisation operations for tests.
type MockFileOptimiser struct {
	CompressOut []byte
	ResizeOut   []byte
	MimeOut     string

	CompressErr error
	ResizeErr   error

	ResizeCalled bool
}

func (m *MockFileOptimiser) Compress(mimeType string, r io.Reader) (io.ReadCloser, string, error) {
	if m.CompressErr != nil {
		return nil, "", m.CompressErr
	}
	return io.NopCloser(bytes.NewReader(m.CompressOut)), m.MimeOut, nil
}

func (m *MockFileOptimiser) Resize(mimeType string, r io.Reader, width, height int) (io.ReadCloser, error) {
	m.ResizeCalled = true
	if m.ResizeErr != nil {
		return nil, m.ResizeErr
	}
	return io.NopCloser(bytes.NewReader(m.ResizeOut)), nil
}
