package testutil

import (
	"context"

	"github.com/fhuszti/medias-ms-go/internal/cache"
	"github.com/fhuszti/medias-ms-go/internal/db"
	workerHandler "github.com/fhuszti/medias-ms-go/internal/handler/worker"
	"github.com/fhuszti/medias-ms-go/internal/optimiser"
	"github.com/fhuszti/medias-ms-go/internal/repository/mariadb"
	"github.com/fhuszti/medias-ms-go/internal/storage"
	"github.com/fhuszti/medias-ms-go/internal/task"
	mediaSvc "github.com/fhuszti/medias-ms-go/internal/usecase/media"
	"github.com/hibiken/asynq"

	"github.com/fhuszti/medias-ms-go/internal/logger"
)

// StartWorker starts an asynq worker processing optimisation tasks.
// It returns a function to gracefully shut down the worker.
func StartWorker(dbConn *db.Database, strg *storage.Strg, redisAddr string) func() {
	repo := mariadb.NewMediaRepository(dbConn.DB)
	fo := optimiser.NewFileOptimiser(optimiser.NewWebPEncoder(), optimiser.NewPDFOptimizer())
	dispatcher := task.NewDispatcher(redisAddr, "")
	ca := cache.NewNoop()
	optimiseSvc := mediaSvc.NewMediaOptimiser(repo, fo, strg, dispatcher, ca)
	resizeSvc := mediaSvc.NewImageResizer(repo, fo, strg, ca)

	mux := asynq.NewServeMux()
	mux.HandleFunc(task.TypeOptimiseMedia, func(ctx context.Context, t *asynq.Task) error {
		p, err := task.ParseOptimiseMediaPayload(t)
		if err != nil {
			return err
		}
		return workerHandler.OptimiseMediaHandler(ctx, p, optimiseSvc)
	})
	mux.HandleFunc(task.TypeResizeImage, func(ctx context.Context, t *asynq.Task) error {
		p, err := task.ParseResizeImagePayload(t)
		if err != nil {
			return err
		}
		return workerHandler.ResizeImageHandler(ctx, p, []int{50, 300}, resizeSvc)
	})

	srv := asynq.NewServer(asynq.RedisClientOpt{Addr: redisAddr}, asynq.Config{Concurrency: 5})
	go func() {
		if err := srv.Run(mux); err != nil {
			logger.Errorf(context.Background(), "worker stopped: %v", err)
		}
	}()

	return func() {
		srv.Shutdown()
	}
}
