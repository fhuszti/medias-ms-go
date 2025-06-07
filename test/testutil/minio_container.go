package testutil

import (
	"context"
	"fmt"
	"github.com/fhuszti/medias-ms-go/internal/storage"
	"log"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
)

type MinIOContainerInfo struct {
	DSN     string
	Strg    *storage.Strg
	Cleanup func()
}

func StartMinIOContainer() (*MinIOContainerInfo, error) {
	const (
		image        = "minio/minio"
		tag          = "latest"
		rootUser     = "minioadmin"
		rootPassword = "minioadmin"
		internalPort = "9000/tcp"
	)

	pool, err := dockertest.NewPool("")
	if err != nil {
		return nil, fmt.Errorf("could not connect to docker: %w", err)
	}

	resource, err := pool.RunWithOptions(&dockertest.RunOptions{
		Repository: image,
		Tag:        tag,
		Env: []string{
			fmt.Sprintf("MINIO_ROOT_USER=%s", rootUser),
			fmt.Sprintf("MINIO_ROOT_PASSWORD=%s", rootPassword),
		},
		Cmd: []string{"server", "/data"},
	}, func(hostConfig *docker.HostConfig) {
		hostConfig.AutoRemove = true
		hostConfig.RestartPolicy = docker.RestartPolicy{Name: "no"}
	})
	if err != nil {
		return nil, fmt.Errorf("could not start minio container: %w", err)
	}

	var endpoint string
	if err := pool.Retry(func() error {
		port := resource.GetPort(internalPort)
		endpoint = fmt.Sprintf("localhost:%s", port)
		client, err := minio.New(endpoint, &minio.Options{
			Creds:  credentials.NewStaticV4(rootUser, rootPassword, ""),
			Secure: false,
		})
		if err != nil {
			return err
		}
		// ListBuckets is a light operation to check health
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_, err = client.ListBuckets(ctx)
		return err
	}); err != nil {
		_ = pool.Purge(resource)
		return nil, fmt.Errorf("minio did not become ready: %w", err)
	}

	strg, err := storage.NewStorage(endpoint, rootUser, rootPassword, false)
	if err != nil {
		pool.Purge(resource)
		return nil, fmt.Errorf("could not create minio client: %w", err)
	}

	port := resource.GetPort(internalPort)
	ci := &MinIOContainerInfo{
		DSN:  fmt.Sprintf("root:secret@(localhost:%s)/testdb?parseTime=true", port),
		Strg: strg,
		Cleanup: func() {
			if err := pool.Purge(resource); err != nil {
				log.Printf("could not purge minio container: %s", err)
			}
		},
	}
	return ci, nil
}
