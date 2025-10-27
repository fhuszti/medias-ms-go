package db

import (
	"database/sql"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

const (
	maxOpenConns    = 20
	maxIdleConns    = 10
	connMaxLifetime = 300 * time.Second
)

// Database holds your SQL connection pool.
type Database struct {
	*sql.DB
}

// New creates, configures, and verifies a MySQL connection pool.
// It returns an error if opening or pinging the database fails.
func New(dsn string) (*Database, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}

	// configure pooling
	db.SetMaxOpenConns(maxOpenConns)
	db.SetMaxIdleConns(maxIdleConns)
	db.SetConnMaxLifetime(connMaxLifetime)

	// verify connectivity
	if err := db.Ping(); err != nil {
		// close the connection pool before returning the ping error
		if cErr := db.Close(); cErr != nil {
			return nil, cErr
		}
		return nil, err
	}
	return &Database{db}, nil
}
