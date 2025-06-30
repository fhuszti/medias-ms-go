package media

import (
	"time"

	"github.com/fhuszti/medias-ms-go/internal/model"
)

// MetadataOutput represents a subset of media metadata returned to clients.
type MetadataOutput struct {
	model.Metadata
	SizeBytes int64  `json:"size_bytes"`
	MimeType  string `json:"mime_type"`
}

// GetMediaOutput describes the result of the GetMedia use case.
type GetMediaOutput struct {
	ValidUntil time.Time            `json:"valid_until"`
	Optimised  bool                 `json:"optimised"`
	URL        string               `json:"url"`
	Metadata   MetadataOutput       `json:"metadata"`
	Variants   model.VariantsOutput `json:"variants"`
}
