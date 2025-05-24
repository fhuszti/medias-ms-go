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

type Variant struct {
	ObjectKey string `json:"object_key"`
	SizeBytes int64  `json:"size_bytes"`
	Width     int    `json:"width"`
	Height    int    `json:"height"`
}

func (v Variant) Value() (driver.Value, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("marshal Variant: %w", err)
	}
	return b, nil
}
func (v *Variant) Scan(src interface{}) error {
	if src == nil {
		*v = Variant{}
		return nil
	}
	data, ok := src.([]byte)
	if !ok {
		return fmt.Errorf("Variant.Scan: expected []byte, got %T", src)
	}
	if err := json.Unmarshal(data, v); err != nil {
		return fmt.Errorf("unmarshal Variant: %w", err)
	}
	return nil
}

type VariantOutput struct {
	URL       string `json:"url"`
	SizeBytes int64  `json:"size_bytes"`
	Width     int    `json:"width"`
	Height    int    `json:"height"`
}

func (v VariantOutput) Value() (driver.Value, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("marshal VariantOutput: %w", err)
	}
	return b, nil
}
func (v *VariantOutput) Scan(src interface{}) error {
	if src == nil {
		*v = VariantOutput{}
		return nil
	}
	data, ok := src.([]byte)
	if !ok {
		return fmt.Errorf("VariantOutput.Scan: expected []byte, got %T", src)
	}
	if err := json.Unmarshal(data, v); err != nil {
		return fmt.Errorf("unmarshal VariantOutput: %w", err)
	}
	return nil
}

type Variants []Variant

func (v Variants) Value() (driver.Value, error) {
	return json.Marshal(v)
}
func (v *Variants) Scan(src interface{}) error {
	if src == nil {
		*v = nil
		return nil
	}
	data, ok := src.([]byte)
	if !ok {
		return fmt.Errorf("Variants.Scan: expected []byte, got %T", src)
	}
	return json.Unmarshal(data, v)
}

type VariantsOutput []VariantOutput

func (v VariantsOutput) Value() (driver.Value, error) {
	return json.Marshal(v)
}
func (v *VariantsOutput) Scan(src interface{}) error {
	if src == nil {
		*v = nil
		return nil
	}
	data, ok := src.([]byte)
	if !ok {
		return fmt.Errorf("VariantsOutput.Scan: expected []byte, got %T", src)
	}
	return json.Unmarshal(data, v)
}
