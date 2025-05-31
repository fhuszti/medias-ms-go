package e2e

import (
	"fmt"
	"github.com/fhuszti/medias-ms-go/internal/storage"
	"github.com/fhuszti/medias-ms-go/test/testutil"
	"os"
	"runtime"
	"testing"
)

var GlobalMinioClient *storage.Strg

func TestMain(m *testing.M) {
	if runtime.GOOS == "windows" {
		// Docker Desktop on Windows typically listens to the named pipe:
		os.Setenv("DOCKER_HOST", "npipe:////./pipe/docker_engine")
	}

	code := func() int {
		dbCleanup, err := setupMariaDB()
		if err != nil {
			fmt.Fprintf(os.Stderr, "DB setup failed: %v\n", err)
			return 1
		}
		defer dbCleanup()

		minioCleanup, err := setupMinIO()
		if err != nil {
			fmt.Fprintf(os.Stderr, "MinIO setup failed: %v\n", err)
			return 1
		}
		defer minioCleanup()

		return m.Run()
	}()

	os.Exit(code)
}

func setupMariaDB() (cleanup func(), err error) {
	if os.Getenv("TEST_DB_DSN") != "" {
		// CI provided it; nothing to clean up
		return func() {}, nil
	}

	mdb, err := testutil.StartMariaDBContainer()
	if err != nil {
		return nil, err
	}

	os.Setenv("TEST_DB_DSN", mdb.DSN)

	return mdb.Cleanup, nil
}

func setupMinIO() (cleanup func(), err error) {
	if os.Getenv("TEST_MINIO_ENDPOINT") != "" {
		// CI path: build the global Strg client
		endpoint := os.Getenv("TEST_MINIO_ENDPOINT")
		access := os.Getenv("TEST_MINIO_ACCESS_KEY")
		secret := os.Getenv("TEST_MINIO_SECRET_KEY")
		useSSL := os.Getenv("TEST_MINIO_USE_SSL") == "true"

		client, err := storage.NewMinioClient(endpoint, access, secret, useSSL)
		if err != nil {
			return nil, err
		}

		GlobalMinioClient = client

		return func() {}, nil
	}

	// local path: start a container
	mi, err := testutil.StartMinIOContainer()
	if err != nil {
		return nil, err
	}

	GlobalMinioClient = mi.Client

	return mi.Cleanup, nil
}
