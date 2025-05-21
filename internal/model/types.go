package model

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

type Metadata struct {
	// generic
	SizeBytes int64  `json:"size_bytes,omitempty"`
	MimeType  string `json:"mime_type,omitempty"`

	// image-specific
	Width  int `json:"width,omitempty"`
	Height int `json:"height,omitempty"`

	// pdf-specific
	PageCount int `json:"page_count,omitempty"`

	// markdown-specific
	WordCount    int64 `json:"word_count,omitempty"`
	HeadingCount int64 `json:"heading_count,omitempty"`
	LinkCount    int64 `json:"link_count,omitempty"`
}

func (m Metadata) Value() (driver.Value, error) {
	b, err := json.Marshal(m)
	if err != nil {
		return nil, fmt.Errorf("marshal Metadata: %w", err)
	}
	return b, nil
}

func (m *Metadata) Scan(src interface{}) error {
	if src == nil {
		*m = Metadata{}
		return nil
	}
	data, ok := src.([]byte)
	if !ok {
		return fmt.Errorf("Metadata.Scan: expected []byte, got %T", src)
	}
	if err := json.Unmarshal(data, m); err != nil {
		return fmt.Errorf("unmarshal Metadata: %w", err)
	}
	return nil
}
