package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/fhuszti/medias-ms-go/internal/config"
	"github.com/fhuszti/medias-ms-go/internal/db"
	workerHandler "github.com/fhuszti/medias-ms-go/internal/handler/worker"
	"github.com/fhuszti/medias-ms-go/internal/optimiser"
	"github.com/fhuszti/medias-ms-go/internal/repository/mariadb"
	"github.com/fhuszti/medias-ms-go/internal/storage"
	"github.com/fhuszti/medias-ms-go/internal/task"
	mediaSvc "github.com/fhuszti/medias-ms-go/internal/usecase/media"
	"github.com/hibiken/asynq"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("❌  Configuration error: %v", err)
	}
	if cfg.RedisAddr == "" {
		log.Fatal("⚠️  REDIS_ADDR must be set to run the worker")
	}

	database := initDb(cfg)
	defer func() {
		if err := database.Close(); err != nil {
			log.Printf("DB close error: %v", err)
		}
	}()

	strg := initStorage(cfg)
	initBuckets(strg, cfg.Buckets)

	repo := mariadb.NewMediaRepository(database.DB)
	fo := optimiser.NewFileOptimiser(optimiser.NewWebPEncoder(), optimiser.NewPDFOptimizer())
	optimiseSvc := mediaSvc.NewMediaOptimiser(repo, fo, strg)

	mux := asynq.NewServeMux()
	mux.HandleFunc(task.TypeOptimiseMedia, func(ctx context.Context, t *asynq.Task) error {
		p, err := task.ParseOptimiseMediaPayload(t)
		if err != nil {
			return err
		}
		return workerHandler.OptimiseMediaHandler(ctx, p, optimiseSvc)
	})

	runWorker(mux, cfg, database)
}

func initDb(cfg *config.Settings) *db.Database {
	log.Println("initialising database...")

	dbCfg := db.MariaDbConfig{
		DSN:             cfg.MariaDBDSN,
		MaxOpenConns:    cfg.MaxOpenConns,
		MaxIdleConns:    cfg.MaxIdleConns,
		ConnMaxLifetime: cfg.ConnMaxLifetime,
	}

	database, err := db.NewFromConfig(dbCfg)
	if err != nil {
		log.Fatalf("❌  Failed to connect to db: %v", err)
	}
	return database
}

func initStorage(cfg *config.Settings) mediaSvc.Storage {
	strg, err := storage.NewStorage(
		cfg.MinioEndpoint,
		cfg.MinioAccessKey,
		cfg.MinioSecretKey,
		cfg.MinioUseSSL,
	)
	if err != nil {
		log.Fatalf("❌  Failed to initialize MinIO client: %v", err)
	}

	return strg
}

func initBuckets(strg mediaSvc.Storage, buckets []string) {
	for _, b := range buckets {
		if err := strg.InitBucket(b); err != nil {
			log.Fatalf("❌  Failed to initialize bucket %q: %v", b, err)
		}
	}
}

func runWorker(mux *asynq.ServeMux, cfg *config.Settings, database *db.Database) {
	srv := asynq.NewServer(asynq.RedisClientOpt{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPassword,
	}, asynq.Config{Concurrency: 10})

	// Run server in background
	go func() {
		if err := srv.Run(mux); err != nil {
			log.Fatalf("❌  Worker failed: %v", err)
		}
	}()
	log.Println("🚀 Worker started")

	// Wait for interrupt signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	<-sigCh
	log.Println("🛑 Shutdown signal received, exiting…")

	// Give Asynq up to 30 sec to finish tasks
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	srv.Shutdown()       // stop accepting new tasks, finish in-flight
	<-shutdownCtx.Done() // either timeout or done

	// Close DB
	if err := database.Close(); err != nil {
		log.Printf("DB close error: %v", err)
	}
	log.Println("✅  Worker gracefully stopped")
}
