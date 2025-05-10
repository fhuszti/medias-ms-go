package testutil

import (
	"database/sql"
	"fmt"
	"os"
	"time"

	"github.com/go-sql-driver/mysql"
	_ "github.com/go-sql-driver/mysql"
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

	cfg, err := mysql.ParseDSN(dsn)
	if err != nil {
		return nil, fmt.Errorf("parse DSN %q: %w", dsn, err)
	}

	origName := cfg.DBName
	cfg.DBName = ""
	rootDSN := cfg.FormatDSN()

	rootDB, err := sql.Open("mysql", rootDSN)
	if err != nil {
		return nil, fmt.Errorf("open root DB: %w", err)
	}

	dbName := fmt.Sprintf("%s_%d", origName, time.Now().UnixNano())
	if _, err := rootDB.Exec("CREATE DATABASE " + dbName); err != nil {
		return nil, err
	}

	cfg.DBName = dbName
	testDSN := cfg.FormatDSN()
	db, err := sql.Open("mysql", testDSN)
	if err != nil {
		rootDB.Exec("DROP DATABASE " + dbName)
		rootDB.Close()
		return nil, fmt.Errorf("open test DB %q: %w", testDSN, err)
	}

	cleanup := func() error {
		err := db.Close()
		if err != nil {
			return err
		}

		if _, err := rootDB.Exec("DROP DATABASE " + dbName); err != nil {
			rootDB.Close()
			return fmt.Errorf("drop database %q: %w", dbName, err)
		}

		return rootDB.Close()
	}

	return &TestDB{DB: db, Cleanup: cleanup}, nil
}
