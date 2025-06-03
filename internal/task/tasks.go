package task

import (
	"encoding/json"
	"fmt"

	"github.com/hibiken/asynq"
)

const TypeOptimiseMedia = "media:optimise"

type OptimiseMediaPayload struct {
	MediaID string `json:"media_id"`
}

// NewOptimiseMediaTask creates an Asynq task for optimising a media by ID.
func NewOptimiseMediaTask(mediaID string) (*asynq.Task, error) {
	p := OptimiseMediaPayload{MediaID: mediaID}
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
