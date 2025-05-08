package model

import (
	"github.com/fhuszti/medias-ms-go/internal/db"
	"time"
)

type MediaStatus string

const (
	MediaStatusPending  MediaStatus = "pending"
	MediaStatusUploaded MediaStatus = "uploaded"
	MediaStatusFailed   MediaStatus = "failed"
)

type Media struct {
	ID             db.UUID     `json:"id"`
	ObjectKey      string      `json:"object_key"`
	MimeType       string      `json:"mime_type"`
	SizeBytes      *int        `json:"size_bytes,omitempty"`
	Status         MediaStatus `json:"status"`
	FailureMessage *string     `json:"failure_message,omitempty"`
	Metadata       *string     `json:"metadata,omitempty"`
	CreatedAt      time.Time   `json:"created_at"`
	UpdatedAt      time.Time   `json:"updated_at"`
}
