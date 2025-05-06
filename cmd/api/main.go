package main

import (
	"context"
	"errors"
	"github.com/fhuszti/medias-ms-go/internal/config"
	"log"
	"strconv"
	"time"

	"github.com/fhuszti/medias-ms-go/internal/db"
	"github.com/fhuszti/medias-ms-go/internal/handler"
	"github.com/fhuszti/medias-ms-go/internal/repository"
	"github.com/fhuszti/medias-ms-go/internal/service"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"net/http"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Configuration error: %v", err)
	}

	database := initDb(cfg)

	r := initRouter()

	npcRepo := repository.NewMariaDBNPCRepository(database.DB)
	npcSvc := service.NewNPCService(npcRepo)
	r.Post("/npcs/create", handler.CreateNPCHandler(npcSvc))

	listenRouter(r, cfg, database)
}

func initDb(cfg *config.Settings) *db.Database {
	dbCfg := db.MariaDbConfig{
		Dsn:             cfg.MariaDBDSN,
		MaxOpenConns:    cfg.MaxOpenConns,
		MaxIdleConns:    cfg.MaxIdleConns,
		ConnMaxLifetime: cfg.ConnMaxLifetime,
	}

	database, err := db.NewFromConfig(dbCfg)
	if err != nil {
		log.Fatalf("Failed to connect to db: %v", err)
	}

	return database
}

func initRouter() *chi.Mux {
	r := chi.NewRouter()

	r.Use(middleware.Logger)

	return r
}

func listenRouter(r *chi.Mux, cfg *config.Settings, database *db.Database) {
	srv := &http.Server{Addr: ":" + strconv.Itoa(cfg.ServerPort), Handler: r}
	go func() {
		log.Printf("Listening on %s", srv.Addr)
		if err := srv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("Listen error: %v", err)
		}
	}()

	// wait for os.Signal (SIGHUP, SIGINT, SIGTERM), then:
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Shutdown error: %v", err)
	}
	err := database.Close()
	if err != nil {
		return
	}
}
