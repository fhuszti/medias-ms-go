package worker

import (
	"context"

	"github.com/fhuszti/medias-ms-go/internal/port"
	"github.com/fhuszti/medias-ms-go/internal/task"
	msuuid "github.com/fhuszti/medias-ms-go/internal/uuid"
	"github.com/fhuszti/medias-ms-go/internal/validation"
	"github.com/google/uuid"

	"github.com/fhuszti/medias-ms-go/internal/logger"
)

// ResizeImageHandler handles a resize-image task.
// It validates the incoming payload and delegates the call to the service.
func ResizeImageHandler(ctx context.Context, p task.ResizeImagePayload, sizes []int, svc port.ImageResizer) error {
	if err := validation.ValidateStruct(p); err != nil {
		logger.Errorf(ctx, "❌  Payload validation failed: %v", err)
		return err
	}

	id := uuid.MustParse(p.ID)
	in := port.ResizeImageInput{ID: msuuid.UUID(id), Sizes: sizes}
	if err := svc.ResizeImage(ctx, in); err != nil {
		logger.Errorf(ctx, "❌  Failed to resize image #%s: %v", id, err)
		return err
	}

	logger.Infof(ctx, "✅  Successfully resized image #%s", id)
	return nil
}
