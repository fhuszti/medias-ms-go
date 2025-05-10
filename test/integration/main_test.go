package integration

import (
	"fmt"
	"github.com/fhuszti/medias-ms-go/internal/storage"
	"github.com/fhuszti/medias-ms-go/test/testutil"
	"os"
	"testing"
)

var GlobalMinioClient *storage.Strg

func TestMain(m *testing.M) {
	mdb, err := testutil.StartMariaDBContainer()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to start MariaDB: %v\n", err)
		os.Exit(1)
	}
	os.Setenv("TEST_DB_DSN", mdb.DSN)

	minio, err := testutil.StartMinIOContainer()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to start MinIO: %v\n", err)
		os.Exit(1)
	}
	GlobalMinioClient = minio.Client

	exitCode := m.Run()

	minio.Cleanup()
	mdb.Cleanup()

	os.Exit(exitCode)
}
