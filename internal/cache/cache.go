package cache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/fhuszti/medias-ms-go/internal/db"
	"github.com/fhuszti/medias-ms-go/internal/usecase/media"
	"github.com/redis/go-redis/v9"
	"hash/crc32"
	"log"
	"time"
)

type Cache struct {
	client *redis.Client
}

// compile-time check: *Cache must satisfy media.Cache
var _ media.Cache = (*Cache)(nil)

func NewCache(addr, password string) *Cache {
	rdb := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       0,
	})

	return &Cache{client: rdb}
}

func (c *Cache) GetMediaDetails(ctx context.Context, id db.UUID) (*media.GetMediaOutput, error) {
	log.Printf("getting entry in cache for media #%s...", id)

	val, err := c.client.Get(ctx, getCacheKey(id.String(), false)).Result()
	if errors.Is(err, redis.Nil) {
		return nil, nil // cache miss
	}
	if err != nil {
		return nil, fmt.Errorf("redis get failed: %w", err)
	}

	var mOut media.GetMediaOutput
	if err := json.Unmarshal([]byte(val), &mOut); err != nil {
		return nil, fmt.Errorf("unmarshal failed: %w", err)
	}

	return &mOut, nil
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

func (c *Cache) SetMediaDetails(ctx context.Context, id db.UUID, mOut *media.GetMediaOutput) {
	log.Printf("creating entry in cache for media #%s, valid until %s...", id, mOut.ValidUntil.Format(time.RFC1123))

	data, err := json.Marshal(mOut)
	if err != nil {
		log.Printf("WARNING: redis set marshal failed: %v", err)
	}

	exp := time.Until(mOut.ValidUntil)

	if err := c.client.Set(ctx, getCacheKey(id.String(), false), data, exp).Err(); err != nil {
		log.Printf("WARNING: redis set failed: %v", err)
	}

	etag := fmt.Sprintf("%08x", crc32.ChecksumIEEE(data))
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

func getCacheKey(id string, isEtag bool) string {
	key := "media:" + id
	if isEtag {
		key = "etag:" + key
	}
	return key
}
