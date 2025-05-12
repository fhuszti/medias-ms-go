package media

const MaxFileSize = 10 * 1024 * 1024 // 10 MB

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

func IsImage(mimeType string) bool {
	return mimeType == "image/png" || mimeType == "image/jpeg" || mimeType == "image/webp"
}

func IsDocument(mimeType string) bool {
	return mimeType == "application/pdf" || mimeType == "text/markdown"
}
