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

// OptimiseMediaHandler handles an optimise-media task.
// It converts the incoming task payload to the input expected by
// the MediaOptimiser service and delegates the call.
func OptimiseMediaHandler(ctx context.Context, p task.OptimiseMediaPayload, svc port.MediaOptimiser) error {
	if err := validation.ValidateStruct(p); err != nil {
		log.Printf("❌  Payload validation failed: %v", err)
		return err
	}

	id := uuid.MustParse(p.ID)

	in := port.OptimiseMediaInput{ID: db.UUID(id)}
	if err := svc.OptimiseMedia(ctx, in); err != nil {
		log.Printf("❌  Failed to optimise media #%s: %v", id, err)
		return err
	}

	log.Printf("✅  Successfully optimised media #%s", id)
	return nil
}
