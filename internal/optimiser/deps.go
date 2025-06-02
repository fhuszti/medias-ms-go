package optimiser

import (
	"image"
	"io"
)

type WebPEncoder interface {
	Encode(img image.Image, quality int, w io.Writer) error
	Decode(r io.Reader) (image.Image, string, error)
}

type PDFOptimizer interface {
	OptimizeFile(inPath, outPath string) error
}
