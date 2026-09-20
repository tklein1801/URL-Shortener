package sqlitestore

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
	"url-shortener/internal/links"
)

func TestRepositoryAndPersistence(t *testing.T) {
	path := filepath.Join(t.TempDir(), "private", "links.db")
	store, err := Open(context.Background(), path, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if err := store.Ping(ctx); err != nil {
		t.Fatal(err)
	}
	var journalMode string
	if err := store.db.QueryRowContext(ctx, "PRAGMA journal_mode").Scan(&journalMode); err != nil || journalMode != "wal" {
		t.Fatalf("journal mode: %q %v", journalMode, err)
	}
	var synchronous, busyTimeout int
	if err := store.db.QueryRowContext(ctx, "PRAGMA synchronous").Scan(&synchronous); err != nil || synchronous != 2 {
		t.Fatalf("synchronous mode: %d %v", synchronous, err)
	}
	if err := store.db.QueryRowContext(ctx, "PRAGMA busy_timeout").Scan(&busyTimeout); err != nil || busyTimeout != 1000 {
		t.Fatalf("busy timeout: %d %v", busyTimeout, err)
	}
	if err := store.Create(ctx, "abcdefgh", "https://example.com"); err != nil {
		t.Fatal(err)
	}
	if err := store.Create(ctx, "abcdefgh", "https://replacement.example"); !errors.Is(err, links.ErrConflict) {
		t.Fatal(err)
	}
	if value, err := store.Get(ctx, "abcdefgh"); err != nil || value != "https://example.com" {
		t.Fatalf("get: %q %v", value, err)
	}
	if _, err := store.Get(ctx, "missing"); !errors.Is(err, links.ErrNotFound) {
		t.Fatal(err)
	}
	if err := store.Create(ctx, "ijklmnop", "https://second.example"); err != nil {
		t.Fatal(err)
	}
	if all, err := store.List(ctx); err != nil || len(all) != 2 {
		t.Fatalf("list: %v %v", all, err)
	}
	if ok, err := store.Delete(ctx, "ijklmnop"); !ok || err != nil {
		t.Fatalf("delete: %v %v", ok, err)
	}
	if ok, err := store.Delete(ctx, "ijklmnop"); ok || err != nil {
		t.Fatalf("delete missing: %v %v", ok, err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	store, err = Open(ctx, path, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	if value, err := store.Get(ctx, "abcdefgh"); err != nil || value != "https://example.com" {
		t.Fatalf("persisted value: %q %v", value, err)
	}
	if runtime.GOOS != "windows" {
		info, err := os.Stat(path)
		if err != nil || info.Mode().Perm() != 0600 {
			t.Fatalf("database mode: %v %v", info, err)
		}
		directory, err := os.Stat(filepath.Dir(path))
		if err != nil || directory.Mode().Perm() != 0700 {
			t.Fatalf("database directory mode: %v %v", directory, err)
		}
	}
	canceled, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := store.Get(canceled, "abcdefgh"); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancellation: %v", err)
	}
}

func TestFailures(t *testing.T) {
	ctx := context.Background()
	parent := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(parent, []byte("not a directory"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := Open(ctx, filepath.Join(parent, "links.db"), time.Second); err == nil {
		t.Fatal("opened database below regular file")
	}
	store, err := Open(ctx, filepath.Join(t.TempDir(), "links.db"), time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Get(ctx, "missing"); !errors.Is(err, links.ErrBackend) || errors.Is(err, links.ErrNotFound) {
		t.Fatalf("closed database error: %v", err)
	}
}
