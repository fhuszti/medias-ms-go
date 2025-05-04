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
		_, err := fmt.Fprintf(os.Stderr, "failed to start MariaDB: %v\n", err)
		if err != nil {
			return
		}
		os.Exit(1)
	}
	defer ci.Cleanup()

	if err := os.Setenv("TEST_DB_DSN", ci.DSN); err != nil {
		return
	}

	os.Exit(m.Run())
}
