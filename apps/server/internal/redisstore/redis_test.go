package redisstore

import (
	"context"
	"errors"
	"fmt"
	"github.com/alicebob/miniredis/v2"
	"testing"
	"time"
	"url-shortener/internal/links"
)

func TestRepository(t *testing.T) {
	redis := miniredis.RunT(t)
	store := New(redis.Addr(), "", 0)
	defer store.Close()
	ctx := context.Background()
	if err := store.Ping(ctx); err != nil {
		t.Fatal(err)
	}
	// Populate enough legacy keys to require multiple SCAN pages.
	for i := 0; i < 350; i++ {
		if err := redis.Set(fmt.Sprintf("old%05d", i), "https://legacy.example"); err != nil {
			t.Fatal(err)
		}
	}
	keys, cursor, err := store.client.Scan(ctx, 0, "*", 100).Result()
	if err != nil || cursor == 0 || len(keys) >= 350 {
		t.Fatalf("test must exercise pagination: %d %d %v", len(keys), cursor, err)
	}
	all, err := store.List(ctx)
	if err != nil || len(all) != 350 {
		t.Fatalf("list %d: %v", len(all), err)
	}
	if err := store.Create(ctx, "old00000", "https://replacement.example"); !errors.Is(err, links.ErrConflict) {
		t.Fatal(err)
	}
	if value, err := store.Get(ctx, "old00000"); err != nil || value != "https://legacy.example" {
		t.Fatalf("overwritten: %s %v", value, err)
	}
	if _, err := store.Get(ctx, "missing"); !errors.Is(err, links.ErrNotFound) {
		t.Fatal(err)
	}
	if err := store.Create(ctx, "abcdefgh", "https://example.com"); err != nil {
		t.Fatal(err)
	}
	if ok, err := store.Delete(ctx, "abcdefgh"); !ok || err != nil {
		t.Fatalf("delete %v %v", ok, err)
	}
	if ok, err := store.Delete(ctx, "abcdefgh"); ok || err != nil {
		t.Fatalf("delete missing %v %v", ok, err)
	}
	redis.Close()
	timeout, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()
	if _, err := store.Get(timeout, "missing"); err == nil || errors.Is(err, links.ErrNotFound) {
		t.Fatalf("outage became missing: %v", err)
	}
}
func TestWrongTypeIsBackendFailure(t *testing.T) {
	redis := miniredis.RunT(t)
	store := New(redis.Addr(), "", 0)
	defer store.Close()
	redis.Lpush("list-key", "value")
	if _, err := store.Get(context.Background(), "list-key"); !errors.Is(err, links.ErrBackend) {
		t.Fatal(err)
	}
	if _, err := store.List(context.Background()); !errors.Is(err, links.ErrBackend) {
		t.Fatal(err)
	}
}
