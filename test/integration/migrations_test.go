package integration

import (
	"testing"
	"time"

	"github.com/fhuszti/medias-ms-go/internal/migration"
	"github.com/fhuszti/medias-ms-go/test/testutil"
	_ "github.com/go-sql-driver/mysql"
)

func TestMigrateUpIntegration(t *testing.T) {
	testDB, err := testutil.SetupTestDB()
	if err != nil {
		t.Fatalf("setup DB: %v", err)
	}
	defer testDB.Cleanup()

	db := testDB.DB

	// Run migrations
	if err := migration.MigrateUp(db); err != nil {
		t.Fatalf("MigrateUp failed: %v", err)
	}

	// Give some time for migration to finalise
	time.Sleep(100 * time.Millisecond)

	// Verify a known table exists
	recs := 0
	err = db.QueryRow("SELECT COUNT(*) FROM medias").Scan(&recs)
	if err != nil {
		t.Fatalf("failed to query migrated table: %v", err)
	}
	// No rows inserted yet, but the query should succeed
	if recs != 0 {
		var id string
		err = db.QueryRow("SELECT id FROM medias").Scan(&id)
		t.Errorf("expected 0 rows in medias after migration, got %d results: %s=", recs, id)
	}
}
