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
	opt := NewFileOptimiser(wEnc, pOpt)

	outRC, mimeType, err := opt.Compress("image/png", strings.NewReader("ignored"))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	defer func(outRC io.ReadCloser) {
		_ = outRC.Close()
	}(outRC)

	out, err := io.ReadAll(outRC)
	if err != nil {
		t.Fatalf("failed to read output: %v", err)
	}
	if string(out) != expected {
		t.Errorf("expected %q, got %q", expected, string(out))
	}
	if mimeType != "image/webp" {
		t.Errorf("expected mime type %q, got %q", "image/webp", mimeType)
	}
}

func TestCompress_ImageMimeType_Conversion(t *testing.T) {
	testCases := []struct {
		inMimeType  string
		outMimeType string
	}{
		{"image/png", "image/webp"},
		{"image/jpeg", "image/webp"},
		{"image/webp", "image/webp"},
	}

	for _, tc := range testCases {
		t.Run(tc.inMimeType, func(t *testing.T) {
			wEnc := &fakeWebPEncoder{returnBytes: []byte("test")}
			pOpt := &fakePDFOptimizer{}
			opt := NewFileOptimiser(wEnc, pOpt)

			_, mimeType, err := opt.Compress(tc.inMimeType, strings.NewReader("test"))
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if mimeType != tc.outMimeType {
				t.Errorf("expected mime type %q, got %q", tc.outMimeType, mimeType)
			}
		})
	}
}

func TestCompress_ImagePath_DecodeError(t *testing.T) {
	wEnc := &fakeWebPEncoder{returnDecodeErr: errors.New("decode failed")}
	pOpt := &fakePDFOptimizer{}
	opt := NewFileOptimiser(wEnc, pOpt)

	reader, newMime, err := opt.Compress("image/jpeg", strings.NewReader("irrelevant"))
	if err != nil {
		t.Fatalf("expected no immediate error, got %v", err)
	}
	defer func(reader io.ReadCloser) {
		_ = reader.Close()
	}(reader)

	_, readErr := io.ReadAll(reader)
	if readErr == nil {
		t.Fatal("expected decode error on read, got nil")
	}
	if !strings.Contains(readErr.Error(), "decode failed") {
		t.Errorf("expected read error to contain 'decode failed', got %q", readErr.Error())
	}
	if newMime != "image/webp" {
		t.Errorf("expected newMimeType 'image/webp' even on decode error, got %q", newMime)
	}
}

func TestCompress_ImagePath_EncodeError(t *testing.T) {
	wEnc := &fakeWebPEncoder{returnEncodeErr: errors.New("encode failed")}
	pOpt := &fakePDFOptimizer{}
	opt := NewFileOptimiser(wEnc, pOpt)

	reader, newMime, err := opt.Compress("image/webp", strings.NewReader("irrelevant"))
	if err != nil {
		t.Fatalf("expected no immediate error, got %v", err)
	}
	defer func(reader io.ReadCloser) {
		_ = reader.Close()
	}(reader)

	_, readErr := io.ReadAll(reader)
	if readErr == nil {
		t.Fatal("expected encode error on read, got nil")
	}
	if !strings.Contains(readErr.Error(), "encode failed") {
		t.Errorf("expected read error to contain 'encode failed', got %q", readErr.Error())
	}
	if newMime != "image/webp" {
		t.Errorf("expected newMimeType 'image/webp' even on encode error, got %q", newMime)
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
	opt := NewFileOptimiser(wEnc, pOpt)

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

	outRC, mimeType, err := opt.Compress("application/pdf", f)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	defer func(outRC io.ReadCloser) {
		_ = outRC.Close()
	}(outRC)

	out, err := io.ReadAll(outRC)
	if err != nil {
		t.Fatalf("failed to read output: %v", err)
	}
	if !strings.HasPrefix(string(out), pdfContent) {
		t.Errorf("expected output to start with %q, got %q", pdfContent, string(out)[:len(pdfContent)])
	}
	if mimeType != "application/pdf" {
		t.Errorf("expected mime type %q, got %q", "application/pdf", mimeType)
	}
}

func TestCompress_PDFPath_Failure(t *testing.T) {
	fakeErr := errors.New("pdf failed")
	wEnc := &fakeWebPEncoder{}
	pOpt := &fakePDFOptimizer{returnErr: fakeErr}
	opt := NewFileOptimiser(wEnc, pOpt)

	tmpIn, err := os.CreateTemp("", "unit_pdf_in_*.pdf")
	if err != nil {
		t.Fatalf("could not create temp input PDF: %v", err)
	}
	defer func(name string) {
		_ = os.Remove(name)
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
		_ = f.Close()
	}(f)

	reader, newMime, err := opt.Compress("application/pdf", f)
	if err != nil {
		t.Fatalf("expected no immediate error, got %v", err)
	}
	defer func(reader io.ReadCloser) {
		_ = reader.Close()
	}(reader)

	_, readErr := io.ReadAll(reader)
	if readErr == nil {
		t.Fatal("expected PDF optimize error on read, got nil")
	}
	if !strings.Contains(readErr.Error(), "pdf failed") {
		t.Errorf("expected read error to contain 'pdf failed', got %q", readErr.Error())
	}
	if newMime != "application/pdf" {
		t.Errorf("expected newMimeType 'application/pdf' even on failure, got %q", newMime)
	}
}

func TestCompress_OtherPath_Success(t *testing.T) {
	wEnc := &fakeWebPEncoder{}
	pOpt := &fakePDFOptimizer{}
	opt := NewFileOptimiser(wEnc, pOpt)

	data := []byte("plain text here")
	outRC, mimeType, err := opt.Compress("text/plain", bytes.NewReader(data))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	defer func(outRC io.ReadCloser) {
		_ = outRC.Close()
	}(outRC)

	out, err := io.ReadAll(outRC)
	if err != nil {
		t.Fatalf("failed to read output: %v", err)
	}
	if !bytes.Equal(out, data) {
		t.Errorf("expected output %q, got %q", data, out)
	}
	if mimeType != "text/plain" {
		t.Errorf("expected mime type %q, got %q", "text/plain", mimeType)
	}
}

func TestCompress_PreserveMimeTypes(t *testing.T) {
	testCases := []struct {
		mimeType string
	}{
		{"application/pdf"},
		{"text/markdown"},
	}

	for _, tc := range testCases {
		t.Run(tc.mimeType, func(t *testing.T) {
			wEnc := &fakeWebPEncoder{}
			pOpt := &fakePDFOptimizer{}
			opt := NewFileOptimiser(wEnc, pOpt)

			_, mimeType, err := opt.Compress(tc.mimeType, strings.NewReader("test"))
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if mimeType != tc.mimeType {
				t.Errorf("expected mime type to be preserved as %q, got %q", tc.mimeType, mimeType)
			}
		})
	}
}

func TestCompress_OtherPath_ReadError(t *testing.T) {
	wEnc := &fakeWebPEncoder{}
	pOpt := &fakePDFOptimizer{}
	opt := NewFileOptimiser(wEnc, pOpt)

	errReader := &errorReader{returnErr: errors.New("read failed")}
	reader, newMime, err := opt.Compress("text/plain", errReader)
	if err != nil {
		t.Fatalf("expected no immediate error, got %v", err)
	}
	defer func(reader io.ReadCloser) {
		_ = reader.Close()
	}(reader)

	_, readErr := io.ReadAll(reader)
	if readErr == nil {
		t.Fatal("expected read error on read, got nil")
	}
	if !strings.Contains(readErr.Error(), "read failed") {
		t.Errorf("expected read error message, got %q", readErr.Error())
	}
	if newMime != "text/plain" {
		t.Errorf("expected newMimeType 'text/plain', got %q", newMime)
	}
}

func TestResize_Image_Success(t *testing.T) {
	const expected = "RESIZED"
	wEnc := &fakeWebPEncoder{returnBytes: []byte(expected)}
	pOpt := &fakePDFOptimizer{}
	opt := NewFileOptimiser(wEnc, pOpt)

	outRC, err := opt.Resize("image/png", strings.NewReader("ignored"), 10, 10)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	defer func() { _ = outRC.Close() }()

	out, err := io.ReadAll(outRC)
	if err != nil {
		t.Fatalf("failed to read output: %v", err)
	}
	if string(out) != expected {
		t.Errorf("expected %q, got %q", expected, string(out))
	}
}

func TestResize_Image_DecodeError(t *testing.T) {
	wEnc := &fakeWebPEncoder{returnDecodeErr: errors.New("dec fail")}
	opt := NewFileOptimiser(wEnc, &fakePDFOptimizer{})

	reader, err := opt.Resize("image/webp", strings.NewReader("irrelevant"), 1, 1)
	if err != nil {
		t.Fatalf("expected no immediate error, got %v", err)
	}
	defer func() { _ = reader.Close() }()

	_, readErr := io.ReadAll(reader)
	if readErr == nil {
		t.Fatal("expected decode error on read, got nil")
	}
	if !strings.Contains(readErr.Error(), "dec fail") {
		t.Errorf("unexpected error %v", readErr)
	}
}

func TestResize_Image_EncodeError(t *testing.T) {
	wEnc := &fakeWebPEncoder{returnEncodeErr: errors.New("enc fail")}
	opt := NewFileOptimiser(wEnc, &fakePDFOptimizer{})

	reader, err := opt.Resize("image/webp", strings.NewReader("irrelevant"), 1, 1)
	if err != nil {
		t.Fatalf("expected no immediate error, got %v", err)
	}
	defer func() { _ = reader.Close() }()

	_, readErr := io.ReadAll(reader)
	if readErr == nil {
		t.Fatal("expected encode error on read, got nil")
	}
	if !strings.Contains(readErr.Error(), "enc fail") {
		t.Errorf("unexpected error %v", readErr)
	}
}

func TestResize_NonImage(t *testing.T) {
	opt := NewFileOptimiser(&fakeWebPEncoder{returnBytes: []byte("NOP")}, &fakePDFOptimizer{})
	data := []byte("plain")
	rc, err := opt.Resize("application/pdf", bytes.NewReader(data), 0, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer func() { _ = rc.Close() }()

	out, err := io.ReadAll(rc)
	if err != nil {
		t.Fatalf("failed to read: %v", err)
	}
	if !bytes.Equal(out, data) {
		t.Errorf("expected %q, got %q", data, out)
	}
}
