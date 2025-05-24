package main

import (
	"github.com/fhuszti/medias-ms-go/internal/config"
	"github.com/fhuszti/medias-ms-go/internal/db"
	"github.com/fhuszti/medias-ms-go/internal/migration"
	_ "github.com/go-sql-driver/mysql"
	"log"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("configuration error: %v", err)
	}

	database, err := initDb(cfg)
	if err != nil {
		log.Fatalf("failed to connect to db: %v", err)
	}
	defer func(database *db.Database) {
		err := database.Close()
		if err != nil {
			return
		}
	}(database)

	if err := migration.MigrateUp(database.DB); err != nil {
		log.Fatalf("migration up failed: %v", err)
	}

	log.Println("migrations applied successfully")
}

func initDb(cfg *config.Settings) (*db.Database, error) {
	dbCfg := db.MariaDbConfig{
		Dsn:             cfg.MariaDBDSN + "&multiStatements=true",
		MaxOpenConns:    cfg.MaxOpenConns,
		MaxIdleConns:    cfg.MaxIdleConns,
		ConnMaxLifetime: cfg.ConnMaxLifetime,
	}

	database, err := db.NewFromConfig(dbCfg)
	if err != nil {
		return nil, err
	}

	return database, nil
}
