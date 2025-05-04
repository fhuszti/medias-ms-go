package testutil

import (
	"database/sql"
	"fmt"
	"os"
	"strings"
	"time"
)

type TestDB struct {
	DB      *sql.DB
	Cleanup func() error
}

func SetupTestDB() (*TestDB, error) {
	dsn := os.Getenv("TEST_DB_DSN")
	if dsn == "" {
		return nil, fmt.Errorf("TEST_DB_DSN env-var not set")
	}

	parts := strings.SplitN(dsn, "/testdb", 2)
	baseDSN := parts[0] + "/"

	rootDB, err := sql.Open("mysql", baseDSN)
	if err != nil {
		return nil, fmt.Errorf("open CI root DB: %w", err)
	}

	dbName := fmt.Sprintf("testdb_%d", time.Now().UnixNano())

	if _, err := rootDB.Exec("CREATE DATABASE " + dbName); err != nil {
		return nil, err
	}

	fullDSN := strings.Replace(dsn, "/testdb", "/"+dbName, 1)
	db, err := sql.Open("mysql", fullDSN)
	if err != nil {
		return nil, fmt.Errorf("open CI DB: %w", err)
	}

	cleanup := func() error {
		err := db.Close()
		if err != nil {
			return err
		}
		if _, err := rootDB.Exec("DROP DATABASE " + dbName); err != nil {
			return err
		}
		return rootDB.Close()
	}

	return &TestDB{DB: db, Cleanup: cleanup}, nil
}
