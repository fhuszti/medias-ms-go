package task

import (
	"encoding/json"
	"fmt"

	"github.com/hibiken/asynq"
)

const TypeCompressMedia = "media:compress"

type CompressMediaPayload struct {
	MediaID string `json:"media_id"`
}

// NewCompressMediaTask creates an Asynq task for compressing a media by ID.
func NewCompressMediaTask(mediaID string) (*asynq.Task, error) {
	p := CompressMediaPayload{MediaID: mediaID}
	data, err := json.Marshal(p)
	if err != nil {
		return nil, fmt.Errorf("could not marshal compress-media payload: %w", err)
	}
	return asynq.NewTask(TypeCompressMedia, data), nil
}

// ParseCompressMediaPayload parses the task payload to CompressMediaPayload.
func ParseCompressMediaPayload(t *asynq.Task) (CompressMediaPayload, error) {
	var p CompressMediaPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return CompressMediaPayload{}, fmt.Errorf("could not unmarshal payload: %w", err)
	}
	return p, nil
}
