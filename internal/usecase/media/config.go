package media

import (
	"fmt"
	"time"
)

const MinFileSize = 1 * 1024         // 1 KB
const MaxFileSize = 10 * 1024 * 1024 // 10 MB

const DownloadUrlTTL = 2 * time.Hour

var AllowedMimeTypes = map[string]bool{
	"image/png":       true,
	"image/jpeg":      true,
	"image/webp":      true,
	"application/pdf": true,
	"text/markdown":   true,
}

func IsMimeTypeAllowed(mimeType string) bool {
	return AllowedMimeTypes[mimeType]
}

func MimeTypeToExtension(mimeType string) (string, error) {
	switch mimeType {
	case "image/png":
		return ".png", nil
	case "image/jpeg":
		return ".jpg", nil
	case "image/webp":
		return ".webp", nil
	case "application/pdf":
		return ".pdf", nil
	case "text/markdown":
		return ".md", nil
	default:
		return "", fmt.Errorf("unsupported mime type: %s", mimeType)
	}
}

func IsImage(mimeType string) bool {
	return mimeType == "image/png" || mimeType == "image/jpeg" || mimeType == "image/webp"
}

func isDocument(mimeType string) bool {
	return IsPdf(mimeType) || IsMarkdown(mimeType)
}

func IsPdf(mimeType string) bool {
	return mimeType == "application/pdf"
}

func IsMarkdown(mimeType string) bool {
	return mimeType == "text/markdown"
}
