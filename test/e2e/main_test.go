package e2e

import (
	"fmt"
	"github.com/fhuszti/medias-ms-go/test/testutil"
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	ci, err := testutil.StartMariaDBContainer()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to start MariaDB: %v\n", err)
		os.Exit(1)
	}
	defer ci.Cleanup()

	if err := os.Setenv("TEST_DB_DSN", ci.DSN); err != nil {
		fmt.Fprintf(os.Stderr, "failed to set TEST_DB_DSN: %v\n", err)
		ci.Cleanup()
		os.Exit(1)
	}

	exitCode := m.Run()

	ci.Cleanup()
	os.Exit(exitCode)
}
