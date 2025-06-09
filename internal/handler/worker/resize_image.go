package worker

import (
	"context"
	"log"

	"github.com/fhuszti/medias-ms-go/internal/db"
	"github.com/fhuszti/medias-ms-go/internal/usecase/media"
	"github.com/fhuszti/medias-ms-go/internal/validation"
	"github.com/google/uuid"
)

// ResizeImagePayload represents the payload for a resize-image task.
type ResizeImagePayload struct {
	ID string `json:"id" validate:"required,uuid"`
}

// ResizeImageHandler handles a resize-image task.
// It validates the incoming payload and delegates the call to the service.
func ResizeImageHandler(ctx context.Context, p ResizeImagePayload, sizes []int, svc media.ImageResizer) error {
	if err := validation.ValidateStruct(p); err != nil {
		log.Printf("❌  Payload validation failed: %v", err)
		return err
	}

	id := uuid.MustParse(p.ID)
	in := media.ResizeImageInput{ID: db.UUID(id), Sizes: sizes}
	if err := svc.ResizeImage(ctx, in); err != nil {
		log.Printf("❌  Failed to resize image #%s: %v", id, err)
		return err
	}

	log.Printf("✅  Successfully resized image #%s", id)
	return nil
}
