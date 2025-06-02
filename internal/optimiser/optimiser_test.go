package optimiser

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/chai2010/webp"
	_ "golang.org/x/image/webp"
)

// helper: generate a 2x2 red PNG, return its bytes.Reader and error
func generatePNG() (io.Reader, error) {
	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	// fill with red
	for x := 0; x < 2; x++ {
		for y := 0; y < 2; y++ {
			img.Set(x, y, color.RGBA{R: 255, A: 255})
		}
	}
	buf := &bytes.Buffer{}
	if err := png.Encode(buf, img); err != nil {
		return nil, err
	}
	return bytes.NewReader(buf.Bytes()), nil
}

// helper: generate a 2x2 red JPEG, return its bytes.Reader and error
func generateJPEG() (io.Reader, error) {
	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	for x := 0; x < 2; x++ {
		for y := 0; y < 2; y++ {
			img.Set(x, y, color.RGBA{G: 255, A: 255})
		}
	}
	buf := &bytes.Buffer{}
	if err := jpeg.Encode(buf, img, &jpeg.Options{Quality: 90}); err != nil {
		return nil, err
	}
	return bytes.NewReader(buf.Bytes()), nil
}

// helper: generate a 2x2 blue WebP, return its bytes.Reader and error
func generateWebP() (io.Reader, error) {
	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	for x := 0; x < 2; x++ {
		for y := 0; y < 2; y++ {
			img.Set(x, y, color.RGBA{B: 255, A: 255})
		}
	}
	buf := &bytes.Buffer{}
	// encode as lossy WebP (though colour is solid, doesn't matter)
	if err := webp.Encode(buf, img, &webp.Options{Quality: 80}); err != nil {
		return nil, err
	}
	return bytes.NewReader(buf.Bytes()), nil
}

func TestCompressPNG(t *testing.T) {
	r, err := generatePNG()
	if err != nil {
		t.Fatalf("failed to generate PNG: %v", err)
	}

	out, err := Compress("image/png", r)
	if err != nil {
		t.Fatalf("Compress(image/png) returned error: %v", err)
	}
	if len(out) == 0 {
		t.Fatal("Compress(image/png) returned empty output")
	}
	// ensure output decodes as WebP
	img, format, err := image.Decode(bytes.NewReader(out))
	if err != nil {
		t.Fatalf("Decoding output as WebP failed: %v", err)
	}
	if format != "webp" {
		t.Errorf("expected format 'webp', got '%s'", format)
	}
	// basic check: decoded image has expected dimensions 2x2
	if img.Bounds().Dx() != 2 || img.Bounds().Dy() != 2 {
		t.Errorf("decoded WebP has wrong dimensions: got %dx%d", img.Bounds().Dx(), img.Bounds().Dy())
	}
}

func TestCompressJPEG(t *testing.T) {
	r, err := generateJPEG()
	if err != nil {
		t.Fatalf("failed to generate JPEG: %v", err)
	}

	out, err := Compress("image/jpeg", r)
	if err != nil {
		t.Fatalf("Compress(image/jpeg) returned error: %v", err)
	}
	if len(out) == 0 {
		t.Fatal("Compress(image/jpeg) returned empty output")
	}
	// ensure output decodes as WebP
	img, format, err := image.Decode(bytes.NewReader(out))
	if err != nil {
		t.Fatalf("Decoding output as WebP failed: %v", err)
	}
	if format != "webp" {
		t.Errorf("expected format 'webp', got '%s'", format)
	}
	if img.Bounds().Dx() != 2 || img.Bounds().Dy() != 2 {
		t.Errorf("decoded WebP has wrong dimensions: got %dx%d", img.Bounds().Dx(), img.Bounds().Dy())
	}
}

func TestCompressWebP(t *testing.T) {
	r, err := generateWebP()
	if err != nil {
		t.Fatalf("failed to generate WebP: %v", err)
	}

	out, err := Compress("image/webp", r)
	if err != nil {
		t.Fatalf("Compress(image/webp) returned error: %v", err)
	}
	if len(out) == 0 {
		t.Fatal("Compress(image/webp) returned empty output")
	}
	// ensure output decodes as WebP
	img, format, err := image.Decode(bytes.NewReader(out))
	if err != nil {
		t.Fatalf("Decoding output as WebP failed: %v", err)
	}
	if format != "webp" {
		t.Errorf("expected format 'webp', got '%s'", format)
	}
	if img.Bounds().Dx() != 2 || img.Bounds().Dy() != 2 {
		t.Errorf("decoded WebP has wrong dimensions: got %dx%d", img.Bounds().Dx(), img.Bounds().Dy())
	}
}

func TestCompressPDF(t *testing.T) {
	f, err := os.Open("testdata/sample.pdf")
	if err != nil {
		t.Fatalf("could not open sample.pdf: %v", err)
	}
	defer func(f *os.File) {
		err := f.Close()
		if err != nil {
			t.Errorf("failed to close file: %v", err)
		}
	}(f)

	out, err := Compress("application/pdf", f)
	if err != nil {
		t.Fatalf("Compress(application/pdf) returned error: %v", err)
	}
	if len(out) == 0 {
		t.Fatal("Compress(application/pdf) returned empty output")
	}
	if !bytes.HasPrefix(out, []byte("%PDF")) {
		t.Errorf("expected output to start with '%%PDF', got %.4s", out[:4])
	}
}

func TestCompressOther(t *testing.T) {
	original := "some plain text not to be changed"
	r := strings.NewReader(original)
	out, err := Compress("text/plain", r)
	if err != nil {
		t.Fatalf("Compress(text/plain) returned error: %v", err)
	}
	result := string(out)
	if result != original {
		t.Errorf("expected output '%s', got '%s'", original, result)
	}
}
