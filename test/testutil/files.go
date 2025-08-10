package testutil

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"os"
	"strings"
	"testing"

	"github.com/chai2010/webp"
	mediaSvc "github.com/fhuszti/medias-ms-go/internal/usecase/media"
)

// GeneratePNG generates a simple RGBA image and encodes it to PNG
func GeneratePNG(t *testing.T, width, height int) []byte {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, color.RGBA{R: 255, G: 255, B: 255, A: 255})
		}
	}
	buf := new(bytes.Buffer)
	if err := png.Encode(buf, img); err != nil {
		t.Fatalf("png encode failed: %v", err)
	}
	// Pad to ensure MinFileSize
	if int64(buf.Len()) < mediaSvc.MinFileSize {
		pad := make([]byte, mediaSvc.MinFileSize-int64(buf.Len()))
		buf.Write(pad)
	}
	return buf.Bytes()
}

func GenerateMarkdown() []byte {
	markdown := strings.Join([]string{
		"# Hello functional Test",
		"## Second Header",
		"## Third Header",
		"This is some content with a [link1](https://example.com).",
		"Another line with a [link2](https://golang.org).",
		strings.Repeat(".", mediaSvc.MinFileSize),
	}, "\n")
	return []byte(markdown)
}

// LoadPDF loads a sample PDF (4 pages)
func LoadPDF(t *testing.T) []byte {
	content, err := os.ReadFile("../testdata/sample.pdf")
	if err != nil {
		t.Fatalf("could not read sample PDF: %v", err)
	}
	return content
}

// GenerateWebP creates a simple WebP image of the given size.
func GenerateWebP(t *testing.T, width, height int) []byte {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, color.RGBA{R: 255, G: 255, B: 255, A: 255})
		}
	}
	buf := new(bytes.Buffer)
	if err := webp.Encode(buf, img, &webp.Options{Quality: 80}); err != nil {
		t.Fatalf("encode webp: %v", err)
	}
	if int64(buf.Len()) < mediaSvc.MinFileSize {
		pad := make([]byte, mediaSvc.MinFileSize-int64(buf.Len()))
		buf.Write(pad)
	}
	return buf.Bytes()
}
