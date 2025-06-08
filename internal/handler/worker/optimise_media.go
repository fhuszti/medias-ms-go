package worker

import (
	"context"
	"log"

	"github.com/fhuszti/medias-ms-go/internal/db"
	"github.com/fhuszti/medias-ms-go/internal/task"
	"github.com/fhuszti/medias-ms-go/internal/usecase/media"
	"github.com/google/uuid"
)

// OptimiseMediaHandler handles an optimise-media task.
// It converts the incoming task payload to the input expected by
// the media.Optimiser service and delegates the call.
func OptimiseMediaHandler(ctx context.Context, p task.OptimiseMediaPayload, svc media.Optimiser) error {
	id, err := uuid.Parse(p.MediaID)
	if err != nil {
		log.Printf("❌  Invalid media ID %q: %v", p.MediaID, err)
		return err
	}

	in := media.OptimiseMediaInput{ID: db.UUID(id)}
	if err := svc.OptimiseMedia(ctx, in); err != nil {
		log.Printf("❌  Failed to optimise media #%s: %v", id, err)
		return err
	}

	log.Printf("✅  Successfully optimised media #%s", id)
	return nil
}
