package testutil

import (
	"bytes"
	mediaSvc "github.com/fhuszti/medias-ms-go/internal/usecase/media"
	"image"
	"image/color"
	"image/png"
	"testing"
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
