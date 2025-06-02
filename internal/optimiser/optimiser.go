package optimiser

import (
	"bytes"
	"fmt"
	"github.com/fhuszti/medias-ms-go/internal/usecase/media"
	_ "golang.org/x/image/webp"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"log"
	"os"
)

type Optimiser struct {
	webpEnc WebPEncoder
	pdfOpt  PDFOptimizer
}

// compile-time check: *Optimiser must satisfy media.Optimiser
var _ media.Optimiser = (*Optimiser)(nil)

func NewOptimiser(webpEnc WebPEncoder, pdfOpt PDFOptimizer) *Optimiser {
	log.Println("initialising optimiser...")
	return &Optimiser{
		webpEnc: webpEnc,
		pdfOpt:  pdfOpt,
	}
}

// Compress takes an input stream (`r`) and its MIME type, then returns a byte slice
// containing the “optimised” version. Behavior:
//   - Images (JPEG, PNG, WebP): always convert to lossy WebP @ quality=80.
//   - PDFs (application/pdf): run pdfcpu.Optimize to strip unused objects.
//   - Everything else (e.g. markdown): read as-is and return raw bytes.
func (o *Optimiser) Compress(mimeType string, r io.Reader) ([]byte, error) {
	switch mimeType {
	case "image/jpeg", "image/png", "image/webp":
		img, _, err := o.webpEnc.Decode(r)
		if err != nil {
			return nil, fmt.Errorf("optimiser: failed to decode image: %w", err)
		}

		buf := &bytes.Buffer{}
		if err := o.webpEnc.Encode(img, 80, buf); err != nil {
			return nil, fmt.Errorf("optimiser: failed to encode WebP: %w", err)
		}
		return buf.Bytes(), nil

	case "application/pdf":
		// Create a temp file to write the incoming PDF
		inFile, err := os.CreateTemp("", "pdf_in_*.pdf")
		if err != nil {
			return nil, fmt.Errorf("optimiser: could not create temp input PDF: %w", err)
		}
		defer func(name string) {
			err := os.Remove(name)
			if err != nil {
				log.Printf("failed to remove in temp file %q: %v", name, err)
			}
		}(inFile.Name())

		// Copy the entire reader into the temp file
		if _, err := io.Copy(inFile, r); err != nil {
			_ = inFile.Close()
			return nil, fmt.Errorf("optimiser: failed to write temp input PDF: %w", err)
		}
		_ = inFile.Close()

		// Create a temp file for the optimised PDF output
		outFile, err := os.CreateTemp("", "pdf_out_*.pdf")
		if err != nil {
			return nil, fmt.Errorf("optimiser: could not create temp output PDF: %w", err)
		}
		_ = outFile.Close()
		defer func(name string) {
			err := os.Remove(name)
			if err != nil {
				log.Printf("failed to remove out temp file %q: %v", name, err)
			}
		}(outFile.Name())

		// Losslessly optimise
		if err := o.pdfOpt.OptimizeFile(inFile.Name(), outFile.Name()); err != nil {
			return nil, fmt.Errorf("optimiser: pdfcpu optimization failed: %w", err)
		}

		// Read back the optimised PDF bytes
		data, err := os.ReadFile(outFile.Name())
		if err != nil {
			return nil, fmt.Errorf("optimiser: failed to read optimized PDF: %w", err)
		}
		return data, nil

	default:
		// For Markdown or any other MIME type, just read & return as-is
		data, err := io.ReadAll(r)
		if err != nil {
			return nil, fmt.Errorf("optimiser: failed to read data: %w", err)
		}
		return data, nil
	}
}

func (o *Optimiser) Resize(mimeType string, r io.Reader, width, height int) ([]byte, error) {
	//TODO implement me
	panic("implement me")
}
