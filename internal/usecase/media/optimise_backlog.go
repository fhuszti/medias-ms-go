package media

import (
	"context"
	"log"
	"time"

	"github.com/fhuszti/medias-ms-go/internal/port"
)

type backlogOptimiserSrv struct {
	repo  port.MediaRepository
	tasks port.TaskDispatcher
}

// compile-time check: *backlogOptimiserSrv must satisfy port.BacklogOptimiser
var _ port.BacklogOptimiser = (*backlogOptimiserSrv)(nil)

// NewBacklogOptimiser constructs a BacklogOptimiser implementation.
func NewBacklogOptimiser(repo port.MediaRepository, tasks port.TaskDispatcher) port.BacklogOptimiser {
	return &backlogOptimiserSrv{repo, tasks}
}

// OptimiseBacklog looks for medias older than one hour that are completed but not optimised
// and enqueues optimisation tasks for them.
func (s *backlogOptimiserSrv) OptimiseBacklog(ctx context.Context) error {
	cutoff := time.Now().Add(-1 * time.Hour)
	ids, err := s.repo.ListUnoptimisedCompletedBefore(ctx, cutoff)
	if err != nil {
		return err
	}

	if len(ids) == 0 {
		log.Printf("no medias found to optimise")
	}

	for _, id := range ids {
		log.Printf("starting optimisation for media #%s", id)
		if err := s.tasks.EnqueueOptimiseMedia(ctx, id); err != nil {
			log.Printf("failed to enqueue optimise task for media #%s: %v", id, err)
		}
	}

	resizeIDs, err := s.repo.ListOptimisedImagesNoVariantsBefore(ctx, cutoff)
	if err != nil {
		return err
	}

	if len(resizeIDs) == 0 {
		log.Printf("no images found to resize")
	}

	for _, id := range resizeIDs {
		log.Printf("starting resize for media #%s", id)
		if err := s.tasks.EnqueueResizeImage(ctx, id); err != nil {
			log.Printf("failed to enqueue resize task for media #%s: %v", id, err)
		}
	}
	return nil
}
