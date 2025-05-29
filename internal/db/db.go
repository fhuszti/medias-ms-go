package db

import (
	"database/sql"
	_ "github.com/go-sql-driver/mysql"
	"time"
)

// Database holds your SQL connection pool.
type Database struct {
	*sql.DB
}

// New creates, configures, and verifies a MySQL connection pool.
// It returns an error if opening or pinging the database fails.
func New(dsn string, maxOpen, maxIdle int, connMaxLifetime time.Duration) (*Database, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}

	// configure pooling
	db.SetMaxOpenConns(maxOpen)
	db.SetMaxIdleConns(maxIdle)
	db.SetConnMaxLifetime(connMaxLifetime)

	// verify connectivity
	if err := db.Ping(); err != nil {
		err := db.Close()
		if err != nil {
			return nil, err
		}
		return nil, err
	}
	return &Database{db}, nil
}

func NewFromConfig(cfg MariaDbConfig) (*Database, error) {
	return New(
		cfg.DSN,
		cfg.MaxOpenConns,
		cfg.MaxIdleConns,
		cfg.ConnMaxLifetime,
	)
}
