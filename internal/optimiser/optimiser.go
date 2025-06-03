package optimiser

import (
	"fmt"
	"github.com/fhuszti/medias-ms-go/internal/usecase/media"
	_ "golang.org/x/image/webp"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"log"
	"os"
)

type FileOptimiser struct {
	webpEnc WebPEncoder
	pdfOpt  PDFOptimizer
}

// compile-time check: *FileOptimiser must satisfy media.FileOptimiser
var _ media.FileOptimiser = (*FileOptimiser)(nil)

func NewFileOptimiser(webpEnc WebPEncoder, pdfOpt PDFOptimizer) *FileOptimiser {
	log.Println("initialising file optimiser...")
	return &FileOptimiser{
		webpEnc: webpEnc,
		pdfOpt:  pdfOpt,
	}
}

// Compress takes an input stream and its MIME type, then returns a byte slice
// containing the “optimised” version. Behavior:
//   - Images (JPEG, PNG, WebP): always convert to lossy WebP @ quality=80.
//   - PDFs (application/pdf): run pdfcpu.Optimize to strip unused objects.
//   - Everything else (e.g. markdown): read as-is and return raw bytes.
func (fo *FileOptimiser) Compress(mimeType string, r io.Reader) (io.ReadCloser, string, error) {
	log.Printf("compressing  file of type %q...", mimeType)

	pr, pw := io.Pipe()

	go func() {
		defer func(pw *io.PipeWriter) {
			_ = pw.Close()
		}(pw)

		switch mimeType {
		case "image/jpeg", "image/png", "image/webp":
			img, _, err := fo.webpEnc.Decode(r)
			if err != nil {
				_ = pw.CloseWithError(fmt.Errorf("optimiser: failed to decode image: %w", err))
				return
			}
			// Re-encode as WebP@80 directly into pw
			if err := fo.webpEnc.Encode(img, 80, pw); err != nil {
				_ = pw.CloseWithError(fmt.Errorf("optimiser: failed to encode WebP: %w", err))
				return
			}

		case "application/pdf":
			// Write incoming PDF to a temp file
			inFile, err := os.CreateTemp("", "pdf_in_*.pdf")
			if err != nil {
				_ = pw.CloseWithError(fmt.Errorf("optimiser: could not create temp input PDF: %w", err))
				return
			}
			inName := inFile.Name()
			if _, err := io.Copy(inFile, r); err != nil {
				_ = inFile.Close()
				_ = pw.CloseWithError(fmt.Errorf("optimiser: failed to write temp input PDF: %w", err))
				return
			}
			_ = inFile.Close()
			defer func(name string) {
				_ = os.Remove(name)
			}(inName)

			// Create a temp output path
			outFile, err := os.CreateTemp("", "pdf_out_*.pdf")
			if err != nil {
				_ = pw.CloseWithError(fmt.Errorf("optimiser: could not create temp output PDF: %w", err))
				return
			}
			outName := outFile.Name()
			_ = outFile.Close()
			defer func(name string) {
				_ = os.Remove(name)
			}(outName)

			// Optimize on disk
			if err := fo.pdfOpt.OptimizeFile(inName, outName); err != nil {
				_ = pw.CloseWithError(fmt.Errorf("optimiser: pdf optimisation failed: %w", err))
				return
			}

			// Stream optimised PDF back into pw
			optimised, err := os.Open(outName)
			if err != nil {
				_ = pw.CloseWithError(fmt.Errorf("optimiser: failed to open optimised PDF: %w", err))
				return
			}
			defer func(optimised *os.File) {
				_ = optimised.Close()
			}(optimised)

			if _, err := io.Copy(pw, optimised); err != nil {
				_ = pw.CloseWithError(fmt.Errorf("optimiser: failed to stream optimised PDF: %w", err))
				return
			}

		default:
			// All other types: pipe raw bytes directly
			if _, err := io.Copy(pw, r); err != nil {
				_ = pw.CloseWithError(fmt.Errorf("optimiser: failed to stream raw data: %w", err))
				return
			}
		}
	}()

	newMimeType := mimeType
	if media.IsImage(mimeType) {
		newMimeType = "image/webp"
	}

	return pr, newMimeType, nil
}

func (fo *FileOptimiser) Resize(mimeType string, r io.Reader, width, height int) ([]byte, error) {
	log.Printf("resizing image of type %q...", mimeType)

	//TODO implement me
	panic("implement me")
}
