package media

import (
	"context"
	"time"

	"github.com/fhuszti/medias-ms-go/internal/port"

	"github.com/fhuszti/medias-ms-go/internal/logger"
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
		logger.Info(ctx, "no medias found to optimise")
	}

	for _, id := range ids {
		logger.Infof(ctx, "starting optimisation for media #%s", id)
		if err := s.tasks.EnqueueOptimiseMedia(ctx, id); err != nil {
			logger.Warnf(ctx, "failed to enqueue optimise task for media #%s: %v", id, err)
		}
	}

	resizeIDs, err := s.repo.ListOptimisedImagesNoVariantsBefore(ctx, cutoff)
	if err != nil {
		return err
	}

	if len(resizeIDs) == 0 {
		logger.Info(ctx, "no images found to resize")
	}

	for _, id := range resizeIDs {
		logger.Infof(ctx, "starting resize for media #%s", id)
		if err := s.tasks.EnqueueResizeImage(ctx, id); err != nil {
			logger.Warnf(ctx, "failed to enqueue resize task for media #%s: %v", id, err)
		}
	}
	return nil
}
