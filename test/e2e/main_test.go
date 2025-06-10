package e2e

import (
	"fmt"
	"github.com/fhuszti/medias-ms-go/internal/storage"
	"github.com/fhuszti/medias-ms-go/test/testutil"
	"os"
	"runtime"
	"testing"
)

var (
	GlobalStrg *storage.Strg
	RedisAddr  string
)

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

		redisCleanup, addr, err := setupRedis()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Redis setup failed: %v\n", err)
			return 1
		}
		RedisAddr = addr
		defer redisCleanup()

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

		strg, err := storage.NewStorage(endpoint, access, secret, useSSL)
		if err != nil {
			return nil, err
		}

		GlobalStrg = strg

		return func() {}, nil
	}

	// local path: start a container
	mi, err := testutil.StartMinIOContainer()
	if err != nil {
		return nil, err
	}

	GlobalStrg = mi.Strg

	return mi.Cleanup, nil
}

func setupRedis() (cleanup func(), addr string, err error) {
	if env := os.Getenv("TEST_REDIS_ADDR"); env != "" {
		return func() {}, env, nil
	}

	rc, err := testutil.StartRedisContainer()
	if err != nil {
		return nil, "", err
	}

	os.Setenv("TEST_REDIS_ADDR", rc.Addr)
	return rc.Cleanup, rc.Addr, nil
}
