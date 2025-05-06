package model

import (
	"github.com/fhuszti/medias-ms-go/internal/db"
	"time"
)

type Media struct {
	ID        db.UUID   `json:"id"`
	ObjectKey string    `json:"object_key"`
	MimeType  string    `json:"mime_type"`
	SizeBytes int       `json:"size_bytes"`
	Status    string    `json:"status"`
	Metadata  string    `json:"metadata"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
