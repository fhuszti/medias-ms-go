# AGENTS.md

## Overview
This repository is a performant microservice for handling file uploads and optimisations. It is written in Go and follows principles of clean architecture and modular design.

## Project Purpose
This microservice is designed to:
- Accept file uploads via presigned URLs.
- Store files in MinIO object storage.
- Optimize media files asynchronously (e.g. lossy WebP conversion, image resizing).
- Persist file metadata in a MariaDB database.
- Remain agnostic of domain logic, making it portable and reusable across projects.

## Key Concepts

### File Lifecycle
- Upload Init: Generate a presigned URL (API: POST /medias/generate_upload_link)
- Staging Upload: Upload to staging bucket using that URL
- Finalization: Frontend calls POST /medias/finalise_upload/{destBucket} to validate and move file

### Post-processing (async):
- Optimize (WebP conversion, etc.)
- Generate resized variants (based on ``IMAGES_SIZES`` env var)

### Supported File Types
- image/jpeg, image/png, image/webp → converted to lossy .webp
- application/pdf → stripped using pdfcpu
- text/markdown → scanned for word count, titles

## Code Structure
- cmd/api/: Entry point for the HTTP API
- cmd/migrate/: Runs SQL migrations
- cmd/optimise-backlog/: CLI to optimise existing media
- cmd/worker/: Background task runner
- internal/cache/: Cache implementation
- internal/config/: Loads and validates configuration
- internal/db/: Database implementation
- internal/handler/api/: HTTP handler layer
- internal/handler/worker/: Async handler layer
- internal/migration/: SQL migrations runner
- internal/model/: Model layer
- internal/optimiser/: File optimiser implementation
- internal/repository/: Database operations
- internal/repository/mariadb/: MariaDB implementation
- internal/storage/: File storage (MinIO) implementation
- internal/task/: Async task definitions
- internal/usecase/: Implementation of the core application logic
- internal/usecase/media/: Media-specific use cases
- internal/validation/: Request validations
- scripts/: Helper scripts (e.g., migration scaffolder)
- test/: Integration and e2e tests
- test/e2e/: End-to-end tests
- test/integration/: Integration tests
- test/testdata/: Sample media files for tests
- test/testutil/: Test utilities

## Make Targets
- make test: Run full test suite
- make unit-tests: Unit tests only
- make functional-tests: Integration and e2e tests
- make migration: Create a new migration file
- make clean: Run linting via golangci-lint

## Environment & Dev Notes
- Docker is used to run dependencies (MinIO, MariaDB)
- Go CLI runs the app and migrations locally
- Redis is optional (used for caching and async processing)
- Missing buckets are auto-created on boot
- staging bucket is always created and used for temporary uploads

## Conventions for Codex
- Services are injected with interface abstractions, never concrete implementations directly.
- External dependencies are hidden behind interface layers.
- Handlers only parse inputs, call services, and return responses—no business logic inside.

## Testing
- Run all tests before finalising a PR, using either ``make test`` or manually:
  - ``go test ./...``
  - then ``cd test/e2e/ && go test ./...``
  - then ``cd test/integration/ && go test ./...``
- All commits must pass lint checks via either ``make clean`` or ``golangci-lint run``
