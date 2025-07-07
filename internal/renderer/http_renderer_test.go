package renderer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"hash/crc32"
	"testing"
	"time"

	"github.com/fhuszti/medias-ms-go/internal/db"
	"github.com/fhuszti/medias-ms-go/internal/mock"
	"github.com/fhuszti/medias-ms-go/internal/port"
)

func TestRenderGetMedia_Cases(t *testing.T) {
	ctx := context.Background()
	id := db.NewUUID()

	t.Run("cache hit", func(t *testing.T) {
		c := &mock.MockCache{Data: []byte(`{"ok":true}`), Etag: "\"1234\""}
		r := NewHTTPRenderer(c)
		getter := &mock.MediaGetter{}

		out, etag, err := r.RenderGetMedia(ctx, getter, id)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if string(out) != string(c.Data) {
			t.Errorf("raw mismatch: got %s want %s", out, c.Data)
		}
		if etag != c.Etag {
			t.Errorf("etag mismatch: got %s want %s", etag, c.Etag)
		}
		if getter.Called {
			t.Error("getter should not be called on cache hit")
		}
		if c.SetMediaCalled || c.SetEtagCalled {
			t.Error("cache should not be set on hit")
		}
	})

	t.Run("cache miss", func(t *testing.T) {
		c := &mock.MockCache{}
		now := time.Now().Add(time.Hour)
		resp := &port.GetMediaOutput{ValidUntil: now}
		getter := &mock.MediaGetter{Out: resp}
		r := NewHTTPRenderer(c)

		out, etag, err := r.RenderGetMedia(ctx, getter, id)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		expected, _ := json.Marshal(resp)
		if string(out) != string(expected) {
			t.Errorf("raw mismatch: got %s want %s", out, expected)
		}
		expEtag := fmt.Sprintf("\"%08x\"", crc32.ChecksumIEEE(expected))
		if etag != expEtag {
			t.Errorf("etag mismatch: got %s want %s", etag, expEtag)
		}
		if !getter.Called {
			t.Error("getter should be called on cache miss")
		}
		if !c.SetMediaCalled || !c.SetEtagCalled {
			t.Error("cache should be written on miss")
		}
		if string(c.Data) != string(expected) {
			t.Errorf("cache data mismatch: got %s want %s", c.Data, expected)
		}
		if c.Etag != expEtag {
			t.Errorf("cached etag mismatch: got %s want %s", c.Etag, expEtag)
		}
	})

	t.Run("getter error", func(t *testing.T) {
		c := &mock.MockCache{}
		g := &mock.MediaGetter{Err: errors.New("fail")}
		r := NewHTTPRenderer(c)

		_, _, err := r.RenderGetMedia(ctx, g, id)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !g.Called {
			t.Error("getter should be called when cache miss")
		}
		if c.SetMediaCalled || c.SetEtagCalled {
			t.Error("cache should not be written on error")
		}
	})

	t.Run("cache error", func(t *testing.T) {
		c := &mock.MockCache{GetMediaErr: errors.New("boom")}
		now := time.Now().Add(time.Hour)
		resp := &port.GetMediaOutput{ValidUntil: now}
		g := &mock.MediaGetter{Out: resp}
		r := NewHTTPRenderer(c)

		_, _, err := r.RenderGetMedia(ctx, g, id)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !g.Called {
			t.Error("getter should be called when cache returns error")
		}
		if !c.SetMediaCalled || !c.SetEtagCalled {
			t.Error("cache should be written when missing due to error")
		}
	})
}
