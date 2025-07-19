package main

import (
	"context"
	"log"

	"github.com/fhuszti/medias-ms-go/internal/config"
	"github.com/fhuszti/medias-ms-go/internal/db"
	"github.com/fhuszti/medias-ms-go/internal/port"
	"github.com/fhuszti/medias-ms-go/internal/repository/mariadb"
	"github.com/fhuszti/medias-ms-go/internal/task"
	mediaSvc "github.com/fhuszti/medias-ms-go/internal/usecase/media"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("❌  Configuration error: %v", err)
	}

	database := initDb(cfg)
	defer func() {
		if err := database.Close(); err != nil {
			log.Printf("DB close error: %v", err)
		}
	}()

	dispatcher := initDispatcher(cfg)
	repo := mariadb.NewMediaRepository(database.DB)

	optimiser := mediaSvc.NewBacklogOptimiser(repo, dispatcher)
	if err := optimiser.OptimiseBacklog(context.Background()); err != nil {
		log.Fatalf("❌  Backlog optimisation failed: %v", err)
	}
	log.Println("✅  Backlog optimisation enqueuing done")
}

func initDb(cfg *config.Settings) *db.Database {
	log.Println("initialising database...")
	
	database, err := db.New(cfg.MariaDBDSN, cfg.MaxOpenConns, cfg.MaxIdleConns, cfg.ConnMaxLifetime)
	if err != nil {
		log.Fatalf("❌  Failed to connect to db: %v", err)
	}
	return database
}

func initDispatcher(cfg *config.Settings) port.TaskDispatcher {
	if cfg.RedisAddr == "" {
		log.Fatalf("❌  Redis not configured: this command requires a running Redis instance")
	}
	return task.NewDispatcher(cfg.RedisAddr, cfg.RedisPassword)
}
