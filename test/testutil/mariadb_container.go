package testutil

import (
	"database/sql"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
	"log"
)

type MariaDBContainerInfo struct {
	DSN     string
	Cleanup func()
}

func StartMariaDBContainer() (*MariaDBContainerInfo, error) {
	const (
		image        = "mariadb"
		tag          = "10.11"
		rootUser     = "root"
		rootPassword = "root"
		internalPort = "3306/tcp"
	)

	pool, err := dockertest.NewPool("")
	if err != nil {
		return nil, fmt.Errorf("could not connect to docker: %w", err)
	}

	resource, err := pool.RunWithOptions(&dockertest.RunOptions{
		Repository: image,
		Tag:        tag,
		Env: []string{
			fmt.Sprintf("MARIADB_ROOT_PASSWORD=%s", rootPassword),
		},
	}, func(hc *docker.HostConfig) {
		hc.AutoRemove = true
		hc.RestartPolicy = docker.RestartPolicy{Name: "no"}
	})
	if err != nil {
		return nil, fmt.Errorf("could not start mariadb container: %w", err)
	}

	var dsn string
	if err := pool.Retry(func() error {
		port := resource.GetPort(internalPort)
		dsn = fmt.Sprintf("%s:%s@(localhost:%s)/mysql?parseTime=true", rootUser, rootPassword, port)
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
	ci := &MariaDBContainerInfo{
		DSN: dsn,
		Cleanup: func() {
			if err := pool.Purge(resource); err != nil {
				log.Printf("could not purge container: %s", err)
			}
		},
	}
	return ci, nil
}
