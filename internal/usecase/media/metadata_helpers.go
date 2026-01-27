package media

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"regexp"
	"strings"

	"github.com/fhuszti/medias-ms-go/internal/model"
	"github.com/ledongthuc/pdf"
	_ "golang.org/x/image/webp"
)

func fillMetadata(mimeType string, file io.Reader) (model.Metadata, error) {
	switch {
	case IsImage(mimeType):
		return fillImageMetadata(file)
	case IsPdf(mimeType):
		return fillPdfMetadata(file)
	case IsMarkdown(mimeType):
		return fillMarkdownMetadata(file)
	default:
		return model.Metadata{}, errors.New("unsupported mime-type")
	}
}

func fillImageMetadata(file io.Reader) (model.Metadata, error) {
	data, err := io.ReadAll(file)
	if err != nil {
		return model.Metadata{}, fmt.Errorf("error reading image data: %w", err)
	}

	cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return model.Metadata{}, fmt.Errorf("error decoding image config: %w", err)
	}

	return model.Metadata{
		Width:  cfg.Width,
		Height: cfg.Height,
	}, nil
}

func fillPdfMetadata(file io.Reader) (model.Metadata, error) {
	data, err := io.ReadAll(file)
	if err != nil {
		return model.Metadata{}, fmt.Errorf("error reading PDF data: %w", err)
	}

	reader, err := pdf.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return model.Metadata{}, fmt.Errorf("error opening pdf reader: %w", err)
	}

	numPages := reader.NumPage()
	return model.Metadata{
		PageCount: numPages,
	}, nil
}

func fillMarkdownMetadata(file io.Reader) (model.Metadata, error) {
	data, err := io.ReadAll(file)
	if err != nil {
		return model.Metadata{}, fmt.Errorf("error reading markdown data: %w", err)
	}

	text := string(data)

	words := strings.Fields(text)

	headingCount := 0
	for _, line := range strings.Split(text, "\n") {
		if strings.HasPrefix(line, "# ") || strings.HasPrefix(line, "##") {
			headingCount++
		}
	}

	linkRe := regexp.MustCompile(`\[[^]]+]\([^)]+\)`)
	links := linkRe.FindAllString(text, -1)

	return model.Metadata{
		WordCount:    int64(len(words)),
		HeadingCount: int64(headingCount),
		LinkCount:    int64(len(links)),
	}, nil
}
