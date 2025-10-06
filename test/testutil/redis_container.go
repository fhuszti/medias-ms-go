package testutil

import (
	"context"
	"fmt"
	"time"

	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
	"github.com/redis/go-redis/v9"

	"github.com/fhuszti/medias-ms-go/internal/logger"
)

type RedisContainerInfo struct {
	Addr    string
	Cleanup func()
}

func StartRedisContainer() (*RedisContainerInfo, error) {
	const (
		image        = "redis"
		tag          = "7"
		internalPort = "6379/tcp"
	)

	pool, err := dockertest.NewPool("")
	if err != nil {
		return nil, fmt.Errorf("could not connect to docker: %w", err)
	}

	resource, err := pool.RunWithOptions(&dockertest.RunOptions{
		Repository: image,
		Tag:        tag,
	}, func(hc *docker.HostConfig) {
		hc.AutoRemove = true
		hc.RestartPolicy = docker.RestartPolicy{Name: "no"}
	})
	if err != nil {
		return nil, fmt.Errorf("could not start redis container: %w", err)
	}

	var addr string
	if err := pool.Retry(func() error {
		addr = fmt.Sprintf("localhost:%s", resource.GetPort(internalPort))
		rdb := redis.NewClient(&redis.Options{Addr: addr})
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		return rdb.Ping(ctx).Err()
	}); err != nil {
		_ = pool.Purge(resource)
		return nil, fmt.Errorf("redis did not become ready: %w", err)
	}

	ci := &RedisContainerInfo{
		Addr: addr,
		Cleanup: func() {
			if err := pool.Purge(resource); err != nil {
				logger.Warnf(context.Background(), "could not purge redis container: %s", err)
			}
		},
	}
	return ci, nil
}
