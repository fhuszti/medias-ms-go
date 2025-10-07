package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/fhuszti/medias-ms-go/internal/cache"
	"github.com/fhuszti/medias-ms-go/internal/config"
	"github.com/fhuszti/medias-ms-go/internal/db"
	"github.com/fhuszti/medias-ms-go/internal/handler/api"
	"github.com/fhuszti/medias-ms-go/internal/logger"
	cMiddleware "github.com/fhuszti/medias-ms-go/internal/middleware"
	"github.com/fhuszti/medias-ms-go/internal/port"
	"github.com/fhuszti/medias-ms-go/internal/renderer"
	"github.com/fhuszti/medias-ms-go/internal/repository/mariadb"
	"github.com/fhuszti/medias-ms-go/internal/storage"
	"github.com/fhuszti/medias-ms-go/internal/task"
	mediaSvc "github.com/fhuszti/medias-ms-go/internal/usecase/media"
	msuuid "github.com/fhuszti/medias-ms-go/internal/uuid"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func main() {
	ctx := context.Background()

	cfg, err := config.Load()
	if err != nil {
		logger.Errorf(ctx, "❌  Configuration error: %v", err)
		os.Exit(1)
	}

	logger.Init()

	database := initDb(ctx, cfg)

	r := initRouter(ctx, cfg.JWTPublicKey)

	strg := initStorage(ctx, cfg)
	initBuckets(ctx, strg, cfg.Buckets)

	mediaRepo := mariadb.NewMediaRepository(database.DB)
	var ca port.Cache
	var dispatcher port.TaskDispatcher
	if cfg.RedisAddr != "" {
		ca = cache.NewCache(cfg.RedisAddr, cfg.RedisPassword)
		dispatcher = task.NewDispatcher(cfg.RedisAddr, cfg.RedisPassword)
		logger.Info(ctx, "✅  Redis cache enabled")
	} else {
		ca = cache.NewNoop()
		dispatcher = task.NewNoopDispatcher()
		logger.Warn(ctx, "⚠️  Redis not configured — caching is disabled")
	}

	uploadLinkGeneratorSvc := mediaSvc.NewUploadLinkGenerator(mediaRepo, strg, msuuid.NewUUID)
	r.Post("/medias/generate_upload_link", api.GenerateUploadLinkHandler(uploadLinkGeneratorSvc))

	uploadFinaliserSvc := mediaSvc.NewUploadFinaliser(mediaRepo, strg, dispatcher)
	r.With(cMiddleware.WithMediaID()).
		Post("/medias/finalise_upload/{id}", api.FinaliseUploadHandler(uploadFinaliserSvc, cfg.Buckets))

	getMediaSvc := mediaSvc.NewMediaGetter(mediaRepo, strg)
	rendererSvc := renderer.NewHTTPRenderer(ca)
	r.With(cMiddleware.WithMediaID()).
		Get("/medias/{id}", api.GetMediaHandler(rendererSvc, getMediaSvc))

	deleteMediaSvc := mediaSvc.NewMediaDeleter(mediaRepo, ca, strg)
	r.With(cMiddleware.WithMediaID()).
		Delete("/medias/{id}", api.DeleteMediaHandler(deleteMediaSvc))

	listenRouter(ctx, r, cfg, database)
}

func initDb(ctx context.Context, cfg *config.Settings) *db.Database {
	logger.Info(ctx, "initialising database...")

	database, err := db.New(cfg.MariaDBDSN, cfg.MaxOpenConns, cfg.MaxIdleConns, cfg.ConnMaxLifetime)
	if err != nil {
		logger.Errorf(ctx, "❌  Failed to connect to db: %v", err)
		os.Exit(1)
	}

	return database
}

func initRouter(ctx context.Context, jwtKey string) *chi.Mux {
	logger.Info(ctx, "initialising router...")

	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(cMiddleware.WithDSTAuth(jwtKey))

	r.NotFound(api.NotFoundHandler())
	r.MethodNotAllowed(api.MethodNotAllowedHandler())

	return r
}

func initStorage(ctx context.Context, cfg *config.Settings) port.Storage {
	strg, err := storage.NewStorage(
		cfg.MinioEndpoint,
		cfg.MinioAccessKey,
		cfg.MinioSecretKey,
		cfg.MinioUseSSL,
	)
	if err != nil {
		logger.Errorf(ctx, "❌  Failed to initialize MinIO client: %v", err)
		os.Exit(1)
	}

	return strg
}

func initBuckets(ctx context.Context, strg port.Storage, buckets []string) {
	for _, b := range buckets {
		if err := strg.InitBucket(b); err != nil {
			logger.Errorf(ctx, "❌  Failed to initialize bucket %q: %v", b, err)
			os.Exit(1)
		}
	}
}

func listenRouter(ctx context.Context, r *chi.Mux, cfg *config.Settings, database *db.Database) {
	srv := &http.Server{Addr: ":" + strconv.Itoa(cfg.ServerPort), Handler: r}

	// start serving
	go func() {
		logger.Infof(ctx, "🚀 API listening on %s", srv.Addr)
		if err := srv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			logger.Errorf(ctx, "❌  Listen error: %v", err)
			os.Exit(1)
		}
	}()

	// block until we get SIGINT/SIGTERM
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Info(ctx, "🛑 Shutdown signal received, exiting…")

	// graceful shutdown with timeout
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Errorf(ctx, "❌  Server shutdown failed: %v", err)
		os.Exit(1)
	}
	logger.Info(ctx, "✅  Server gracefully stopped")

	if err := database.Close(); err != nil {
		logger.Errorf(ctx, "DB close error: %v", err)
		os.Exit(1)
	}
}
