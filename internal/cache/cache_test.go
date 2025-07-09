package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"hash/crc32"
	"strings"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/fhuszti/medias-ms-go/internal/model"
	"github.com/fhuszti/medias-ms-go/internal/port"
	msuuid "github.com/fhuszti/medias-ms-go/internal/uuid"
	"github.com/redis/go-redis/v9"
)

func makeTestCache(t *testing.T) (*Cache, *miniredis.Miniredis) {
	// spin up in-memory Redis
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis.Run: %v", err)
	}
	// point the real client at it
	rdb := redis.NewClient(&redis.Options{
		Addr:     mr.Addr(),
		Password: "",
		DB:       0,
	})
	return &Cache{client: rdb}, mr
}

func TestGetSetDeleteMediaDetails(t *testing.T) {
	c, mr := makeTestCache(t)
	ctx := context.Background()

	// prepare a sample GetMediaOutput
	id := msuuid.NewUUID()
	out := &port.GetMediaOutput{
		ValidUntil: time.Now().Add(2 * time.Minute),
		Optimised:  false,
		URL:        "https://example.com/download/" + id.String(),
		Metadata: port.MetadataOutput{
			Metadata:  model.Metadata{PageCount: 3},
			SizeBytes: 12345,
			MimeType:  "application/pdf",
		},
		Variants: nil,
	}

	// 1) Cache miss
	gotBytes, err := c.GetMediaDetails(ctx, id)
	if err != nil {
		t.Fatalf("GetMediaDetails miss: %v", err)
	}
	if gotBytes != nil {
		t.Errorf("GetMediaDetails miss: got %v; want nil", gotBytes)
	}

	// 2) Set + Get
	raw, _ := json.Marshal(out)
	c.SetMediaDetails(ctx, id, raw, out.ValidUntil)
	wantETag := fmt.Sprintf("%08x", crc32.ChecksumIEEE(raw))
	c.SetEtagMediaDetails(ctx, id, wantETag, out.ValidUntil)
	// check TTL in Redis ≈ 2m
	if ttl := mr.TTL(getCacheKey(id.String(), false)); ttl < time.Minute*1 || ttl > time.Minute*2+time.Second {
		t.Errorf("redis TTL = %v; want ~2m", ttl)
	}
	if ttl := mr.TTL(getCacheKey(id.String(), true)); ttl < time.Minute*1 || ttl > time.Minute*2+time.Second {
		t.Errorf("etag TTL = %v; want ~2m", ttl)
	}
	if et, err := mr.Get(getCacheKey(id.String(), true)); err != nil {
		t.Fatalf("etag get error: %v", err)
	} else if et != wantETag {
		t.Errorf("etag value = %q; want %q", et, wantETag)
	}
	gotBytes, err = c.GetMediaDetails(ctx, id)
	if err != nil {
		t.Fatalf("GetMediaDetails hit: %v", err)
	}
	if gotBytes == nil {
		t.Fatal("GetMediaDetails hit: got nil; want non-nil")
	}
	var got port.GetMediaOutput
	if err := json.Unmarshal(gotBytes, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.URL != out.URL || got.Optimised != out.Optimised ||
		got.Metadata.SizeBytes != out.Metadata.SizeBytes ||
		got.Metadata.MimeType != out.Metadata.MimeType {
		t.Errorf("roundtrip mismatch: got %+v; want %+v", got, out)
	}

	// 3) Delete + miss again
	if err := c.DeleteMediaDetails(ctx, id); err != nil {
		t.Fatalf("DeleteMediaDetails: %v", err)
	}
	if gotBytes, _ := c.GetMediaDetails(ctx, id); gotBytes != nil {
		t.Errorf("after delete, GetMediaDetails = %v; want nil", gotBytes)
	}
}

func TestGetMediaDetails_BadJSON(t *testing.T) {
	c, mr := makeTestCache(t)
	ctx := context.Background()
	id := msuuid.NewUUID()

	// inject invalid JSON into Redis
	if err := mr.Set(getCacheKey(id.String(), false), "{ not valid json }"); err != nil {
		t.Fatalf("Manually set cache: %v", err)
	}

	gotBytes, err := c.GetMediaDetails(ctx, id)
	if gotBytes != nil {
		t.Errorf("Expected nil on bad JSON, got %v", gotBytes)
	}
	if err == nil || !strings.Contains(err.Error(), "unmarshal failed") {
		t.Errorf("Expected unmarshal failed error, got %v", err)
	}
}

func TestGetMediaDetails_RedisError(t *testing.T) {
	c, mr := makeTestCache(t)
	ctx := context.Background()
	id := msuuid.NewUUID()

	// Simulate Redis unreachable
	mr.Close()

	gotBytes, err := c.GetMediaDetails(ctx, id)
	if gotBytes != nil {
		t.Errorf("Expected nil on Redis error, got %v", gotBytes)
	}
	if err == nil || !strings.Contains(err.Error(), "redis get failed") {
		t.Errorf("Expected redis get failed error, got %v", err)
	}
}

func TestDeleteMediaDetails_RedisError(t *testing.T) {
	c, mr := makeTestCache(t)
	ctx := context.Background()
	id := msuuid.NewUUID()

	// Simulate Redis unreachable before Delete
	mr.Close()

	err := c.DeleteMediaDetails(ctx, id)
	if err == nil || !strings.Contains(err.Error(), "redis del failed") {
		t.Errorf("Expected redis del failed error, got %v", err)
	}
}

func TestDeleteEtagMediaDetails(t *testing.T) {
	c, _ := makeTestCache(t)
	ctx := context.Background()

	id := msuuid.NewUUID()
	out := &port.GetMediaOutput{ValidUntil: time.Now().Add(2 * time.Minute)}
	raw, _ := json.Marshal(out)
	etag := fmt.Sprintf("%08x", crc32.ChecksumIEEE(raw))
	c.SetMediaDetails(ctx, id, raw, out.ValidUntil)
	c.SetEtagMediaDetails(ctx, id, etag, out.ValidUntil)

	if err := c.DeleteEtagMediaDetails(ctx, id); err != nil {
		t.Fatalf("DeleteEtagMediaDetails: %v", err)
	}
	if got, _ := c.GetEtagMediaDetails(ctx, id); got != "" {
		t.Errorf("expected empty string after delete, got %q", got)
	}
}

func TestDeleteEtagMediaDetails_RedisError(t *testing.T) {
	c, mr := makeTestCache(t)
	ctx := context.Background()
	id := msuuid.NewUUID()

	mr.Close()
	err := c.DeleteEtagMediaDetails(ctx, id)
	if err == nil || !strings.Contains(err.Error(), "redis del failed") {
		t.Errorf("Expected redis del failed error, got %v", err)
	}
}

func TestGetCacheKey_Etag(t *testing.T) {
	id := msuuid.NewUUID().String()
	if got := getCacheKey(id, true); got != "etag:media:"+id {
		t.Errorf("getCacheKey(true) = %q; want %q", got, "etag:media:"+id)
	}
	if got := getCacheKey(id, false); got != "media:"+id {
		t.Errorf("getCacheKey() = %q; want %q", got, "media:"+id)
	}
}

func TestGetEtagMediaDetails(t *testing.T) {
	c, mr := makeTestCache(t)
	ctx := context.Background()

	id := msuuid.NewUUID()
	if got, err := c.GetEtagMediaDetails(ctx, id); err != nil {
		t.Fatalf("initial miss err: %v", err)
	} else if got != "" {
		t.Errorf("expected empty string on miss, got %q", got)
	}
	out := &port.GetMediaOutput{ValidUntil: time.Now().Add(2 * time.Minute)}

	raw, _ := json.Marshal(out)
	want := fmt.Sprintf("%08x", crc32.ChecksumIEEE(raw))
	c.SetMediaDetails(ctx, id, raw, out.ValidUntil)
	c.SetEtagMediaDetails(ctx, id, want, out.ValidUntil)

	got, err := c.GetEtagMediaDetails(ctx, id)
	if err != nil {
		t.Fatalf("GetEtagMediaDetails: %v", err)
	}
	if got != want {
		t.Errorf("GetEtagMediaDetails = %q; want %q", got, want)
	}

	mr.Close()
	if _, err := c.GetEtagMediaDetails(ctx, id); err == nil || !strings.Contains(err.Error(), "redis get failed") {
		t.Errorf("expected redis get failed error, got %v", err)
	}
}
