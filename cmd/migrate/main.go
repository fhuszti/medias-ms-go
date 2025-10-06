package main

import (
	"context"
	"os"

	"github.com/fhuszti/medias-ms-go/internal/config"
	"github.com/fhuszti/medias-ms-go/internal/db"
	"github.com/fhuszti/medias-ms-go/internal/migration"
	_ "github.com/go-sql-driver/mysql"

	"github.com/fhuszti/medias-ms-go/internal/logger"
)

func main() {
	ctx := context.Background()

	cfg, err := config.Load()
	if err != nil {
		logger.Errorf(ctx, "❌  Configuration error: %v", err)
		os.Exit(1)
	}

	database, err := initDb(cfg)
	if err != nil {
		logger.Errorf(ctx, "❌  Failed to connect to db: %v", err)
		os.Exit(1)
	}
	defer func(database *db.Database) {
		err := database.Close()
		if err != nil {
			return
		}
	}(database)

	if err := migration.MigrateUp(database.DB); err != nil {
		logger.Errorf(ctx, "❌  Migration up failed: %v", err)
		os.Exit(1)
	}

	logger.Info(ctx, "✅  Migrations applied successfully")
}

func initDb(cfg *config.Settings) (*db.Database, error) {
	database, err := db.New(cfg.MariaDBDSN+"&multiStatements=true", cfg.MaxOpenConns, cfg.MaxIdleConns, cfg.ConnMaxLifetime)
	if err != nil {
		return nil, err
	}

	return database, nil
}
