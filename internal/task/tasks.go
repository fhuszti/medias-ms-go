package task

import (
	"encoding/json"
	"fmt"

	"github.com/hibiken/asynq"
)

const TypeOptimiseMedia = "media:optimise"
const TypeResizeImage = "image:resize"

type OptimiseMediaPayload struct {
	ID string `json:"id" validate:"required,uuid"`
}

type ResizeImagePayload struct {
	ID string `json:"id" validate:"required,uuid"`
}

// NewOptimiseMediaTask creates an Asynq task for optimising a media by ID.
func NewOptimiseMediaTask(id string) (*asynq.Task, error) {
	p := OptimiseMediaPayload{ID: id}
	data, err := json.Marshal(p)
	if err != nil {
		return nil, fmt.Errorf("could not marshal optimise-media payload: %w", err)
	}
	return asynq.NewTask(TypeOptimiseMedia, data), nil
}

// ParseOptimiseMediaPayload parses the task payload to OptimiseMediaPayload.
func ParseOptimiseMediaPayload(t *asynq.Task) (OptimiseMediaPayload, error) {
	var p OptimiseMediaPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return OptimiseMediaPayload{}, fmt.Errorf("could not unmarshal payload: %w", err)
	}
	return p, nil
}

// NewResizeImageTask creates an Asynq task for resizing an image.
func NewResizeImageTask(id string) (*asynq.Task, error) {
	p := ResizeImagePayload{ID: id}
	data, err := json.Marshal(p)
	if err != nil {
		return nil, fmt.Errorf("could not marshal resize-image payload: %w", err)
	}
	return asynq.NewTask(TypeResizeImage, data), nil
}

// ParseResizeImagePayload parses the task payload to ResizeImagePayload.
func ParseResizeImagePayload(t *asynq.Task) (ResizeImagePayload, error) {
	var p ResizeImagePayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return ResizeImagePayload{}, fmt.Errorf("could not unmarshal payload: %w", err)
	}
	return p, nil
}
