package port

import "io"

// FileOptimiser defines file optimisation operations, such as compressing and resizing files.
type FileOptimiser interface {
	Compress(mimeType string, r io.Reader) (io.ReadCloser, string, error)
	Resize(mimeType string, r io.Reader, width, height int) (io.ReadCloser, error)
}
