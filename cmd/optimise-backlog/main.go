package main

import (
	"context"
	"os"

	"github.com/fhuszti/medias-ms-go/internal/config"
	"github.com/fhuszti/medias-ms-go/internal/db"
	"github.com/fhuszti/medias-ms-go/internal/port"
	"github.com/fhuszti/medias-ms-go/internal/repository/mariadb"
	"github.com/fhuszti/medias-ms-go/internal/task"
	mediaSvc "github.com/fhuszti/medias-ms-go/internal/usecase/media"

	"github.com/fhuszti/medias-ms-go/internal/logger"
)

func main() {
	ctx := context.Background()

	cfg, err := config.Load()
	if err != nil {
		logger.Errorf(ctx, "❌  Configuration error: %v", err)
		os.Exit(1)
	}

	database := initDb(cfg)
	defer func() {
		if err := database.Close(); err != nil {
			logger.Warnf(ctx, "DB close error: %v", err)
		}
	}()

	dispatcher := initDispatcher(cfg)
	repo := mariadb.NewMediaRepository(database.DB)

	optimiser := mediaSvc.NewBacklogOptimiser(repo, dispatcher)
	if err := optimiser.OptimiseBacklog(ctx); err != nil {
		logger.Errorf(ctx, "❌  Backlog optimisation failed: %v", err)
		os.Exit(1)
	}
	logger.Info(ctx, "✅  Backlog optimisation enqueuing done")
}

func initDb(cfg *config.Settings) *db.Database {
	ctx := context.Background()
	logger.Info(ctx, "initialising database...")

	database, err := db.New(cfg.MariaDBDSN)
	if err != nil {
		logger.Errorf(ctx, "❌  Failed to connect to db: %v", err)
		os.Exit(1)
	}
	return database
}

func initDispatcher(cfg *config.Settings) port.TaskDispatcher {
	if cfg.RedisAddr == "" {
		logger.Error(context.Background(), "❌  Redis not configured: this command requires a running Redis instance")
		os.Exit(1)
	}
	return task.NewDispatcher(cfg.RedisAddr, cfg.RedisPassword)
}
