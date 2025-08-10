package main

import (
	"log"

	"github.com/fhuszti/medias-ms-go/internal/config"
	"github.com/fhuszti/medias-ms-go/internal/db"
	"github.com/fhuszti/medias-ms-go/internal/migration"
	_ "github.com/go-sql-driver/mysql"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("❌  Configuration error: %v", err)
	}

	database, err := initDb(cfg)
	if err != nil {
		log.Fatalf("❌  Failed to connect to db: %v", err)
	}
	defer func(database *db.Database) {
		err := database.Close()
		if err != nil {
			return
		}
	}(database)

	if err := migration.MigrateUp(database.DB); err != nil {
		log.Fatalf("❌  Migration up failed: %v", err)
	}

	log.Println("✅  Migrations applied successfully")
}

func initDb(cfg *config.Settings) (*db.Database, error) {
	database, err := db.New(cfg.MariaDBDSN+"&multiStatements=true", cfg.MaxOpenConns, cfg.MaxIdleConns, cfg.ConnMaxLifetime)
	if err != nil {
		return nil, err
	}

	return database, nil
}
