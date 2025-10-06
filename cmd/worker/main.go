package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/fhuszti/medias-ms-go/internal/cache"
	"github.com/fhuszti/medias-ms-go/internal/config"
	"github.com/fhuszti/medias-ms-go/internal/db"
	workerHandler "github.com/fhuszti/medias-ms-go/internal/handler/worker"
	"github.com/fhuszti/medias-ms-go/internal/optimiser"
	"github.com/fhuszti/medias-ms-go/internal/port"
	"github.com/fhuszti/medias-ms-go/internal/repository/mariadb"
	"github.com/fhuszti/medias-ms-go/internal/storage"
	"github.com/fhuszti/medias-ms-go/internal/task"
	mediaSvc "github.com/fhuszti/medias-ms-go/internal/usecase/media"
	"github.com/hibiken/asynq"

	"github.com/fhuszti/medias-ms-go/internal/logger"
)

func main() {
	ctx := context.Background()

	cfg, err := config.Load()
	if err != nil {
		logger.Errorf(ctx, "❌  Configuration error: %v", err)
		os.Exit(1)
	}
	if cfg.RedisAddr == "" {
		logger.Error(ctx, "⚠️  REDIS_ADDR must be set to run the worker")
		os.Exit(1)
	}

	database := initDb(cfg)
	defer func() {
		if err := database.Close(); err != nil {
			logger.Warnf(ctx, "DB close error: %v", err)
		}
	}()

	strg := initStorage(cfg)
	initBuckets(strg, cfg.Buckets)

	repo := mariadb.NewMediaRepository(database.DB)
	fo := optimiser.NewFileOptimiser(optimiser.NewWebPEncoder(), optimiser.NewPDFOptimizer())
	dispatcher := task.NewDispatcher(cfg.RedisAddr, cfg.RedisPassword)
	ca := cache.NewCache(cfg.RedisAddr, cfg.RedisPassword)
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
		return workerHandler.ResizeImageHandler(ctx, p, cfg.ImagesSizes, resizeSvc)
	})

	runWorker(ctx, mux, cfg, database)
}

func initDb(cfg *config.Settings) *db.Database {
	ctx := context.Background()
	logger.Info(ctx, "initialising database...")

	database, err := db.New(cfg.MariaDBDSN, cfg.MaxOpenConns, cfg.MaxIdleConns, cfg.ConnMaxLifetime)
	if err != nil {
		logger.Errorf(ctx, "❌  Failed to connect to db: %v", err)
		os.Exit(1)
	}
	return database
}

func initStorage(cfg *config.Settings) port.Storage {
	strg, err := storage.NewStorage(
		cfg.MinioEndpoint,
		cfg.MinioAccessKey,
		cfg.MinioSecretKey,
		cfg.MinioUseSSL,
	)
	if err != nil {
		logger.Errorf(context.Background(), "❌  Failed to initialize MinIO client: %v", err)
		os.Exit(1)
	}

	return strg
}

func initBuckets(strg port.Storage, buckets []string) {
	for _, b := range buckets {
		if err := strg.InitBucket(b); err != nil {
			logger.Errorf(context.Background(), "❌  Failed to initialize bucket %q: %v", b, err)
			os.Exit(1)
		}
	}
}

func runWorker(ctx context.Context, mux *asynq.ServeMux, cfg *config.Settings, database *db.Database) {
	srv := asynq.NewServer(asynq.RedisClientOpt{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPassword,
	}, asynq.Config{Concurrency: 10})

	// Run server in background
	go func() {
		if err := srv.Run(mux); err != nil {
			logger.Errorf(context.Background(), "❌  Worker failed: %v", err)
			os.Exit(1)
		}
	}()
	logger.Info(ctx, "🚀 Worker started")

	// Wait for interrupt signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	<-sigCh
	logger.Info(ctx, "🛑 Shutdown signal received, exiting…")

	// Give Asynq up to 30 sec to finish tasks
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	srv.Shutdown()       // stop accepting new tasks, finish in-flight
	<-shutdownCtx.Done() // either timeout or done

	// Close DB
	if err := database.Close(); err != nil {
		logger.Warnf(ctx, "DB close error: %v", err)
	}
	logger.Info(ctx, "✅  Worker gracefully stopped")
}
