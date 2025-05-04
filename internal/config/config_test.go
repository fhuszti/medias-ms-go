package config

import (
	"os"
	"testing"
	"time"
)

func TestLoad_Success(t *testing.T) {
	// Switch to a temp directory to avoid loading a real .env
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("could not get working directory: %v", err)
	}
	tmpDir := t.TempDir()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("could not chdir to temp dir: %v", err)
	}
	defer func() {
		if err := os.Chdir(origDir); err != nil {
			t.Fatalf("could not chdir back to original dir: %v", err)
		}
	}()

	// Set all required environment variables
	reqs := map[string]string{
		"MARIADB_DSN":               "user:pass@tcp(localhost:3306)/db",
		"MARIADB_MAX_OPEN_CONN":     "10",
		"MARIADB_MAX_IDLE_CONNS":    "5",
		"MARIADB_CONN_MAX_LIFETIME": "30",
		"SERVER_PORT":               "8080",
	}
	for k, v := range reqs {
		t.Setenv(k, v)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if cfg.MariaDBDSN != reqs["MARIADB_DSN"] {
		t.Errorf("MariaDBDSN: expected %q, got %q", reqs["MARIADB_DSN"], cfg.MariaDBDSN)
	}
	if cfg.MaxOpenConns != 10 {
		t.Errorf("MaxOpenConns: expected %d, got %d", 10, cfg.MaxOpenConns)
	}
	if cfg.MaxIdleConns != 5 {
		t.Errorf("MaxIdleConns: expected %d, got %d", 5, cfg.MaxIdleConns)
	}
	if cfg.ConnMaxLifetime != 30*time.Second {
		t.Errorf("ConnMaxLifetime: expected %v, got %v", 30*time.Second, cfg.ConnMaxLifetime)
	}
	if cfg.ServerPort != 8080 {
		t.Errorf("ServerPort: expected %d, got %d", 8080, cfg.ServerPort)
	}
}

func TestLoad_MissingRequiredVars(t *testing.T) {
	cases := []struct {
		missingKey string
		wantErr    string
	}{
		{"MARIADB_DSN", "MARIADB_DSN is required"},
		{"MARIADB_MAX_OPEN_CONN", "MARIADB_MAX_OPEN_CONN is required"},
		{"MARIADB_MAX_IDLE_CONNS", "MARIADB_MAX_IDLE_CONNS is required"},
		{"MARIADB_CONN_MAX_LIFETIME", "MARIADB_CONN_MAX_LIFETIME is required"},
		{"SERVER_PORT", "SERVER_PORT is required"},
	}

	for _, tc := range cases {
		t.Run(tc.missingKey, func(t *testing.T) {
			// Isolate .env loading
			origDir, err := os.Getwd()
			if err != nil {
				t.Fatalf("could not get working directory: %v", err)
			}
			tmpDir := t.TempDir()
			if err := os.Chdir(tmpDir); err != nil {
				t.Fatalf("could not chdir to temp dir: %v", err)
			}
			defer func() {
				if err := os.Chdir(origDir); err != nil {
					t.Fatalf("could not chdir back to original dir: %v", err)
				}
			}()

			// Set all except the missing key
			reqs := map[string]string{
				"MARIADB_DSN":               "user:pass@tcp(localhost:3306)/db",
				"MARIADB_MAX_OPEN_CONN":     "10",
				"MARIADB_MAX_IDLE_CONNS":    "5",
				"MARIADB_CONN_MAX_LIFETIME": "30",
				"SERVER_PORT":               "8080",
			}
			for k, v := range reqs {
				if k == tc.missingKey {
					if err := os.Unsetenv(k); err != nil {
						t.Fatalf("could not unset key %s in env: %v", k, err)
					}
				} else {
					t.Setenv(k, v)
				}
			}

			cfg, err := Load()
			if err == nil {
				t.Fatalf("expected error for missing %s, got nil", tc.missingKey)
			}
			if err.Error() != tc.wantErr {
				t.Errorf("error = %q; want %q", err.Error(), tc.wantErr)
			}
			if cfg != nil {
				t.Errorf("expected cfg nil on error, got %#v", cfg)
			}
		})
	}
}
