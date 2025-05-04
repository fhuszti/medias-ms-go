package testutil

import (
	"database/sql"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
	"log"
)

type ContainerInfo struct {
	DSN     string
	Cleanup func()
}

func StartMariaDBContainer() (*ContainerInfo, error) {
	pool, err := dockertest.NewPool("")
	if err != nil {
		return nil, fmt.Errorf("connect to docker: %w", err)
	}

	resource, err := pool.RunWithOptions(&dockertest.RunOptions{
		Repository: "mariadb",
		Tag:        "10.11",
		Env: []string{
			"MARIADB_ROOT_PASSWORD=secret",
		},
	}, func(hc *docker.HostConfig) {
		hc.AutoRemove = true
		hc.RestartPolicy = docker.RestartPolicy{Name: "no"}
	})
	if err != nil {
		return nil, fmt.Errorf("run mariadb: %w", err)
	}

	if err := pool.Retry(func() error {
		port := resource.GetPort("3306/tcp")
		dsn := fmt.Sprintf("root:secret@(localhost:%s)/mysql?parseTime=true", port)
		db, err := sql.Open("mysql", dsn)
		if err != nil {
			return err
		}
		defer func(db *sql.DB) {
			err := db.Close()
			if err != nil {
				return
			}
		}(db)
		return db.Ping()
	}); err != nil {
		_ = pool.Purge(resource)
		return nil, fmt.Errorf("mariadb did not become ready: %w", err)
	}

	port := resource.GetPort("3306/tcp")
	ci := &ContainerInfo{
		DSN: fmt.Sprintf("root:secret@(localhost:%s)/testdb?parseTime=true", port),
		Cleanup: func() {
			if err := pool.Purge(resource); err != nil {
				log.Printf("could not purge container: %s", err)
			}
		},
	}
	return ci, nil
}
