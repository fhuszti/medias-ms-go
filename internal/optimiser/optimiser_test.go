package optimiser

import (
	"bytes"
	"errors"
	"image"
	"image/color"
	"image/draw"
	"io"
	"os"
	"strings"
	"testing"

	_ "golang.org/x/image/webp"
)

type fakeWebPEncoder struct {
	returnDecodeErr error
	returnEncodeErr error
	returnBytes     []byte
}

func (f *fakeWebPEncoder) Decode(r io.Reader) (image.Image, string, error) {
	if f.returnDecodeErr != nil {
		return nil, "", f.returnDecodeErr
	}
	// Just return a 1x1 image
	img := image.NewRGBA(image.Rect(0, 0, 1, 1))
	draw.Draw(img, img.Bounds(), &image.Uniform{C: color.RGBA{R: 255, A: 255}}, image.Point{}, draw.Src)
	return img, "png", nil
}

func (f *fakeWebPEncoder) Encode(img image.Image, quality int, w io.Writer) error {
	if f.returnEncodeErr != nil {
		return f.returnEncodeErr
	}
	if f.returnBytes != nil {
		_, _ = w.Write(f.returnBytes)
	}
	return nil
}

type fakePDFOptimizer struct {
	returnErr error
}

func (f *fakePDFOptimizer) OptimizeFile(inPath, outPath string) error {
	if f.returnErr != nil {
		return f.returnErr
	}
	// Simply read inPath and write to outPath
	data, err := os.ReadFile(inPath)
	if err != nil {
		return err
	}
	return os.WriteFile(outPath, data, 0644)
}

// errorReader always returns an error on Read.
type errorReader struct {
	returnErr error
}

func (e *errorReader) Read(p []byte) (int, error) {
	return 0, e.returnErr
}

func TestCompress_ImagePath_Success(t *testing.T) {
	const expected = "HELLOIMG"
	wEnc := &fakeWebPEncoder{returnBytes: []byte(expected)}
	pOpt := &fakePDFOptimizer{}
	opt := NewOptimiser(wEnc, pOpt)

	out, err := opt.Compress("image/png", strings.NewReader("ignored"))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if string(out) != expected {
		t.Errorf("expected %q, got %q", expected, string(out))
	}
}

func TestCompress_ImagePath_DecodeError(t *testing.T) {
	wEnc := &fakeWebPEncoder{returnDecodeErr: errors.New("decode failed")}
	pOpt := &fakePDFOptimizer{}
	opt := NewOptimiser(wEnc, pOpt)

	_, err := opt.Compress("image/jpeg", strings.NewReader("irrelevant"))
	if err == nil {
		t.Fatal("expected decode error, got nil")
	}
	if !strings.Contains(err.Error(), "decode failed") {
		t.Errorf("expected decode error message, got %q", err.Error())
	}
}

func TestCompress_ImagePath_EncodeError(t *testing.T) {
	wEnc := &fakeWebPEncoder{returnEncodeErr: errors.New("encode failed")}
	pOpt := &fakePDFOptimizer{}
	opt := NewOptimiser(wEnc, pOpt)

	_, err := opt.Compress("image/webp", strings.NewReader("irrelevant"))
	if err == nil {
		t.Fatal("expected encode error, got nil")
	}
	if !strings.Contains(err.Error(), "encode failed") {
		t.Errorf("expected encode error message, got %q", err.Error())
	}
}

func TestCompress_PDFPath_Success(t *testing.T) {
	tmpIn, err := os.CreateTemp("", "unit_pdf_in_*.pdf")
	if err != nil {
		t.Fatalf("could not create temp input PDF: %v", err)
	}
	defer func(name string) {
		err := os.Remove(name)
		if err != nil {
			t.Logf("failed to remove temp file %q: %v", name, err)
		}
	}(tmpIn.Name())
	const pdfContent = "%PDF-UNIT-TEST"
	if _, err := tmpIn.WriteString(pdfContent); err != nil {
		_ = tmpIn.Close()
		t.Fatalf("could not write to temp PDF: %v", err)
	}
	_ = tmpIn.Close()

	wEnc := &fakeWebPEncoder{}
	pOpt := &fakePDFOptimizer{}
	opt := NewOptimiser(wEnc, pOpt)

	f, err := os.Open(tmpIn.Name())
	if err != nil {
		t.Fatalf("could not open temp input PDF: %v", err)
	}
	defer func(f *os.File) {
		err := f.Close()
		if err != nil {
			t.Logf("failed to close temp file: %v", err)
		}
	}(f)

	out, err := opt.Compress("application/pdf", f)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !strings.HasPrefix(string(out), pdfContent) {
		t.Errorf("expected output to start with %q, got %q", pdfContent, string(out)[:len(pdfContent)])
	}
}

func TestCompress_PDFPath_Failure(t *testing.T) {
	fakeErr := errors.New("pdf failed")
	wEnc := &fakeWebPEncoder{}
	pOpt := &fakePDFOptimizer{returnErr: fakeErr}
	opt := NewOptimiser(wEnc, pOpt)

	tmpIn, err := os.CreateTemp("", "unit_pdf_in_*.pdf")
	if err != nil {
		t.Fatalf("could not create temp input PDF: %v", err)
	}
	defer func(name string) {
		err := os.Remove(name)
		if err != nil {
			t.Logf("failed to remove temp file %q: %v", name, err)
		}
	}(tmpIn.Name())
	if _, err := tmpIn.WriteString("%PDF-UNIT-FAIL"); err != nil {
		_ = tmpIn.Close()
		t.Fatalf("could not write to temp PDF: %v", err)
	}
	_ = tmpIn.Close()

	f, err := os.Open(tmpIn.Name())
	if err != nil {
		t.Fatalf("could not open temp input PDF: %v", err)
	}
	defer func(f *os.File) {
		err := f.Close()
		if err != nil {
			t.Logf("failed to close temp file: %v", err)
		}
	}(f)

	_, err = opt.Compress("application/pdf", f)
	if err == nil {
		t.Fatal("expected PDF optimize error, got nil")
	}
	if !strings.Contains(err.Error(), "pdf failed") {
		t.Errorf("expected error to contain %q, got %q", fakeErr.Error(), err.Error())
	}
}

func TestCompress_OtherPath_Success(t *testing.T) {
	wEnc := &fakeWebPEncoder{}
	pOpt := &fakePDFOptimizer{}
	opt := NewOptimiser(wEnc, pOpt)

	data := []byte("plain text here")
	out, err := opt.Compress("text/plain", bytes.NewReader(data))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !bytes.Equal(out, data) {
		t.Errorf("expected output %q, got %q", data, out)
	}
}

func TestCompress_OtherPath_ReadError(t *testing.T) {
	wEnc := &fakeWebPEncoder{}
	pOpt := &fakePDFOptimizer{}
	opt := NewOptimiser(wEnc, pOpt)

	errReader := &errorReader{returnErr: errors.New("read failed")}
	_, err := opt.Compress("text/plain", errReader)
	if err == nil {
		t.Fatal("expected read error, got nil")
	}
	if !strings.Contains(err.Error(), "read failed") {
		t.Errorf("expected read error message, got %q", err.Error())
	}
}
