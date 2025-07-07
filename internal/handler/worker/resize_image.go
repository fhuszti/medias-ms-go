package worker

import (
	"context"
	"github.com/fhuszti/medias-ms-go/internal/port"
	"log"

	"github.com/fhuszti/medias-ms-go/internal/db"
	"github.com/fhuszti/medias-ms-go/internal/task"
	"github.com/fhuszti/medias-ms-go/internal/validation"
	"github.com/google/uuid"
)

// ResizeImageHandler handles a resize-image task.
// It validates the incoming payload and delegates the call to the service.
func ResizeImageHandler(ctx context.Context, p task.ResizeImagePayload, sizes []int, svc port.ImageResizer) error {
	if err := validation.ValidateStruct(p); err != nil {
		log.Printf("❌  Payload validation failed: %v", err)
		return err
	}

	id := uuid.MustParse(p.ID)
	in := port.ResizeImageInput{ID: db.UUID(id), Sizes: sizes}
	if err := svc.ResizeImage(ctx, in); err != nil {
		log.Printf("❌  Failed to resize image #%s: %v", id, err)
		return err
	}

	log.Printf("✅  Successfully resized image #%s", id)
	return nil
}
