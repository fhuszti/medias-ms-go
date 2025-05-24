package model

import (
	"github.com/fhuszti/medias-ms-go/internal/db"
	"time"
)

type MediaStatus string

const (
	MediaStatusPending   MediaStatus = "pending"
	MediaStatusCompleted MediaStatus = "completed"
	MediaStatusFailed    MediaStatus = "failed"
)

type Media struct {
	ID               db.UUID     `json:"id"`
	ObjectKey        string      `json:"object_key"`
	Bucket           string      `json:"bucket"`
	OriginalFilename string      `json:"original_filename"`
	MimeType         *string     `json:"mime_type,omitempty"`
	SizeBytes        *int64      `json:"size_bytes,omitempty"`
	Status           MediaStatus `json:"status"`
	Optimised        bool        `json:"optimised"`
	FailureMessage   *string     `json:"failure_message,omitempty"`
	Metadata         Metadata    `json:"metadata"`
	Variants         Variants    `json:"variants"`
	CreatedAt        time.Time   `json:"created_at"`
	UpdatedAt        time.Time   `json:"updated_at"`
}
