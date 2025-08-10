package optimiser

import (
	"image"
	"io"

	"github.com/chai2010/webp"
	"github.com/pdfcpu/pdfcpu/pkg/api"
)

type WebPEncoder interface {
	Encode(img image.Image, quality int, w io.Writer) error
	Decode(r io.Reader) (image.Image, string, error)
}

type PDFOptimizer interface {
	OptimizeFile(inPath, outPath string) error
}

type webPEncoder struct{}

func NewWebPEncoder() WebPEncoder {
	return &webPEncoder{}
}

func (e *webPEncoder) Decode(r io.Reader) (image.Image, string, error) {
	img, format, err := image.Decode(r)
	return img, format, err
}

func (e *webPEncoder) Encode(img image.Image, quality int, w io.Writer) error {
	opts := &webp.Options{Quality: float32(quality)}
	return webp.Encode(w, img, opts)
}

type pdfOptimizer struct{}

func NewPDFOptimizer() PDFOptimizer {
	return &pdfOptimizer{}
}

func (p *pdfOptimizer) OptimizeFile(inPath, outPath string) error {
	return api.OptimizeFile(inPath, outPath, nil)
}
