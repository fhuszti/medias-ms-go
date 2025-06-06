package db

import (
	"testing"
	"time"
)

// TestNew_PingError ensures that ping failures are propagated
// even when closing the connection succeeds.
func TestNew_PingError(t *testing.T) {
	// Use an unreachable DSN to trigger ping error quickly
	dsn := "invalid:invalid@tcp(127.0.0.1:0)/dbname"
	db, err := New(dsn, 1, 1, time.Second)
	if err == nil {
		if db != nil {
			db.Close()
		}
		t.Fatalf("expected error, got nil")
	}
}
