package worker

import (
	"context"
	"log"

	"github.com/fhuszti/medias-ms-go/internal/db"
	"github.com/fhuszti/medias-ms-go/internal/usecase/media"
	"github.com/google/uuid"
)

// ResizeImagePayload represents the payload for a resize-image task.
type ResizeImagePayload struct {
	MediaID string `json:"media_id"`
	Sizes   []int  `json:"sizes"`
}

// ResizeImageHandler handles a resize-image task.
// It validates the incoming payload and delegates the call to the service.
func ResizeImageHandler(ctx context.Context, p ResizeImagePayload, svc media.ImageResizer) error {
	id, err := uuid.Parse(p.MediaID)
	if err != nil {
		log.Printf("❌  Invalid media ID %q: %v", p.MediaID, err)
		return err
	}

	in := media.ResizeImageInput{ID: db.UUID(id), Sizes: p.Sizes}
	if err := svc.ResizeImage(ctx, in); err != nil {
		log.Printf("❌  Failed to resize image #%s: %v", id, err)
		return err
	}

	log.Printf("✅  Successfully resized image #%s", id)
	return nil
}
