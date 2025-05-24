package migration

import (
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/mysql"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"log"
	"sort"
	"strconv"
	"strings"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

func MigrateUp(db *sql.DB) error {
	src, err := iofs.New(migrationsFS, "migrations")
	if err != nil {
		return fmt.Errorf("could not create source driver: %v", err)
	}

	driver, err := mysql.WithInstance(db, &mysql.Config{})
	if err != nil {
		return fmt.Errorf("could not create migration driver: %v", err)
	}

	m, err := migrate.NewWithInstance("iofs", src, "mysql", driver)
	if err != nil {
		return fmt.Errorf("failed to initialize migration: %v", err)
	}

	err = m.Up()
	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		// if it's a dirty error, roll back to the previous version and retry
		var dirtyErr migrate.ErrDirty
		log.Printf("hello: %v", errors.As(err, &dirtyErr))
		if errors.As(err, &dirtyErr) {
			prev, err := getPreviousVersionFromDirty(dirtyErr.Version)
			if err != nil {
				return err
			}
			log.Printf("database dirty at version %d, forcing back to %d", dirtyErr.Version, prev)
			if ferr := m.Force(int(prev)); ferr != nil {
				return fmt.Errorf("failed to force to version %d: %w", prev, ferr)
			}
			// retry Up() once more
			if err2 := m.Up(); err2 != nil && !errors.Is(err2, migrate.ErrNoChange) {
				return fmt.Errorf("migration up failed after force: %w", err2)
			}
			return nil
		}
		// some other error
		return fmt.Errorf("migration up failed: %w", err)
	}

	return nil
}

func getPreviousVersionFromDirty(dirtyVersion int) (uint64, error) {
	// read available migration versions from embedded FS
	entries, readErr := migrationsFS.ReadDir("migrations")
	if readErr != nil {
		return 0, fmt.Errorf("dirty at %d but failed to read migrations directory: %w", dirtyVersion, readErr)
	}
	// collect and sort version numbers
	var versions []uint64
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasSuffix(name, ".up.sql") {
			// filename format: <version>_<description>.up.sql
			parts := strings.SplitN(name, "_", 2)
			verStr := parts[0]
			v, parseErr := strconv.ParseUint(verStr, 10, 64)
			if parseErr != nil {
				continue
			}
			versions = append(versions, v)
		}
	}
	sort.Slice(versions, func(i, j int) bool { return versions[i] < versions[j] })
	// find the previous version before the dirty one
	var prev uint64
	for i, v := range versions {
		if v == uint64(dirtyVersion) && i > 0 {
			prev = versions[i-1]
			break
		}
	}
	if prev == 0 {
		return 0, fmt.Errorf("could not determine previous version before %d", dirtyVersion)
	}

	return prev, nil
}
