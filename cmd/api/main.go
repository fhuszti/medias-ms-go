package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/fhuszti/medias-ms-go/internal/handler"
	"github.com/fhuszti/medias-ms-go/internal/storage"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/fhuszti/medias-ms-go/internal/config"
	"github.com/fhuszti/medias-ms-go/internal/db"
	mediaHandler "github.com/fhuszti/medias-ms-go/internal/handler/media"
	"github.com/fhuszti/medias-ms-go/internal/repository/mariadb"
	mediaSvc "github.com/fhuszti/medias-ms-go/internal/usecase/media"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Configuration error: %v", err)
	}

	database := initDb(cfg)

	r := initRouter()

	storages := initBucketStorages(cfg)
	mediaRepo := mariadb.NewMediaRepository(database.DB)

	uploadLinkGeneratorSvc := mediaSvc.NewUploadLinkGenerator(mediaRepo, storages["staging"], db.NewUUID)
	r.Post("/medias/generate_upload_link", mediaHandler.GenerateUploadLinkHandler(uploadLinkGeneratorSvc))

	var getStrgFromBucket mediaSvc.StorageGetter = func(bucket string) (mediaSvc.Storage, error) {
		st, ok := storages[bucket]
		if !ok {
			return nil, fmt.Errorf("bucket %q is not configured", bucket)
		}
		return st, nil
	}
	uploadFinaliserSvc := mediaSvc.NewUploadFinaliser(mediaRepo, storages["staging"], getStrgFromBucket)
	r.With(mediaHandler.WithDestBucket(cfg.Buckets)).
		Post("/medias/finalise_upload/{destBucket}", mediaHandler.FinaliseUploadHandler(uploadFinaliserSvc))

	listenRouter(r, cfg, database)
}

func initDb(cfg *config.Settings) *db.Database {
	log.Println("initialising database...")

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
	log.Println("initialising router...")

	r := chi.NewRouter()

	r.Use(middleware.Logger)

	r.NotFound(handler.NotFoundHandler())
	r.MethodNotAllowed(handler.MethodNotAllowedHandler())

	return r
}

func initBucketStorages(cfg *config.Settings) map[string]mediaSvc.Storage {
	strg, err := storage.NewMinioClient(
		cfg.MinioEndpoint,
		cfg.MinioAccessKey,
		cfg.MinioSecretKey,
		cfg.MinioUseSSL,
	)
	if err != nil {
		log.Fatalf("Failed to initialize MinIO client: %v", err)
	}

	storages := make(map[string]mediaSvc.Storage, len(cfg.Buckets))
	for _, b := range cfg.Buckets {
		storages[b], err = strg.WithBucket(b)
		if err != nil {
			log.Fatalf("Failed to initialize bucket '%s': %v", b, err)
		}
	}

	_, ok := storages["staging"]
	if !ok {
		log.Fatal("You need a bucket named 'staging'")
	}

	return storages
}

func listenRouter(r *chi.Mux, cfg *config.Settings, database *db.Database) {
	srv := &http.Server{Addr: ":" + strconv.Itoa(cfg.ServerPort), Handler: r}

	// start serving
	go func() {
		log.Printf("🚀 API listening on %s", srv.Addr)
		if err := srv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("Listen error: %v", err)
		}
	}()

	// block until we get SIGINT/SIGTERM
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("🛑 shutdown signal received, exiting…")

	// graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server shutdown failed: %v", err)
	}
	log.Println("✅ server gracefully stopped")

	if err := database.Close(); err != nil {
		log.Printf("DB close error: %v", err)
	}
}
