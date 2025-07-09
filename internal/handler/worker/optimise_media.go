package worker

import (
	"context"
	"log"

	"github.com/fhuszti/medias-ms-go/internal/port"
	"github.com/fhuszti/medias-ms-go/internal/task"
	msuuid "github.com/fhuszti/medias-ms-go/internal/uuid"
	"github.com/fhuszti/medias-ms-go/internal/validation"
	"github.com/google/uuid"
)

// OptimiseMediaHandler handles an optimise-media task.
// It converts the incoming task payload to the input expected by
// the MediaOptimiser service and delegates the call.
func OptimiseMediaHandler(ctx context.Context, p task.OptimiseMediaPayload, svc port.MediaOptimiser) error {
	if err := validation.ValidateStruct(p); err != nil {
		log.Printf("❌  Payload validation failed: %v", err)
		return err
	}

	id := uuid.MustParse(p.ID)

	if err := svc.OptimiseMedia(ctx, msuuid.UUID(id)); err != nil {
		log.Printf("❌  Failed to optimise media #%s: %v", id, err)
		return err
	}

	log.Printf("✅  Successfully optimised media #%s", id)
	return nil
}
