package cache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/fhuszti/medias-ms-go/internal/db"
	"github.com/fhuszti/medias-ms-go/internal/port"
	"github.com/redis/go-redis/v9"
)

type Cache struct {
	client *redis.Client
}

// compile-time check: *Cache must satisfy port.Cache
var _ port.Cache = (*Cache)(nil)

func NewCache(addr, password string) *Cache {
	rdb := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       0,
	})

	return &Cache{client: rdb}
}

func (c *Cache) GetMediaDetails(ctx context.Context, id db.UUID) ([]byte, error) {
	log.Printf("getting entry in cache for media #%s...", id)

	val, err := c.client.Get(ctx, getCacheKey(id.String(), false)).Result()
	if errors.Is(err, redis.Nil) {
		return nil, nil // cache miss
	}
	if err != nil {
		return nil, fmt.Errorf("redis get failed: %w", err)
	}
	data := []byte(val)
	if !json.Valid(data) {
		return nil, fmt.Errorf("unmarshal failed: invalid JSON")
	}
	return data, nil
}

func (c *Cache) GetEtagMediaDetails(ctx context.Context, id db.UUID) (string, error) {
	log.Printf("getting etag in cache for media #%s...", id)

	val, err := c.client.Get(ctx, getCacheKey(id.String(), true)).Result()
	if errors.Is(err, redis.Nil) {
		return "", nil // cache miss
	}
	if err != nil {
		return "", fmt.Errorf("redis get failed: %w", err)
	}

	return val, nil
}

func (c *Cache) SetMediaDetails(ctx context.Context, id db.UUID, data []byte, validUntil time.Time) {
	log.Printf("creating entry in cache for media #%s, valid until %s...", id, validUntil.Format(time.RFC1123))
	exp := time.Until(validUntil)

	if err := c.client.Set(ctx, getCacheKey(id.String(), false), data, exp).Err(); err != nil {
		log.Printf("WARNING: redis set failed: %v", err)
	}
}

func (c *Cache) SetEtagMediaDetails(ctx context.Context, id db.UUID, etag string, validUntil time.Time) {
	log.Printf("creating etag in cache for media #%s, valid until %s...", id, validUntil.Format(time.RFC1123))
	exp := time.Until(validUntil)

	if err := c.client.Set(ctx, getCacheKey(id.String(), true), etag, exp).Err(); err != nil {
		log.Printf("WARNING: redis set failed: %v", err)
	}
}

func (c *Cache) DeleteMediaDetails(ctx context.Context, id db.UUID) error {
	log.Printf("deleting entry in cache for media #%s...", id)

	if err := c.client.Del(ctx, getCacheKey(id.String(), false)).Err(); err != nil {
		return fmt.Errorf("redis del failed: %w", err)
	}
	return nil
}

func (c *Cache) DeleteEtagMediaDetails(ctx context.Context, id db.UUID) error {
	log.Printf("deleting etag in cache for media #%s...", id)

	if err := c.client.Del(ctx, getCacheKey(id.String(), true)).Err(); err != nil {
		return fmt.Errorf("redis del failed: %w", err)
	}
	return nil
}

func getCacheKey(id string, isEtag bool) string {
	key := "media:" + id
	if isEtag {
		key = "etag:" + key
	}
	return key
}
