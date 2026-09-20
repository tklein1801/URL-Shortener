// Package sqlitestore persists shortened links in a local SQLite database.
package sqlitestore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
	"url-shortener/internal/links"

	_ "modernc.org/sqlite"
)

type Store struct{ db *sql.DB }

var _ links.Repository = (*Store)(nil)

func Open(ctx context.Context, path string, busyTimeout time.Duration) (_ *Store, err error) {
	if path == "" {
		return nil, errors.New("SQLite path must not be empty")
	}
	if busyTimeout <= 0 {
		return nil, errors.New("SQLite busy timeout must be positive")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return nil, fmt.Errorf("create SQLite directory: %w", err)
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open SQLite database: %w", err)
	}
	defer func() {
		if err != nil {
			_ = db.Close()
		}
	}()
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	store := &Store{db: db}
	if err = db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("connect to SQLite database: %w", err)
	}
	milliseconds := busyTimeout.Milliseconds()
	if milliseconds < 1 {
		milliseconds = 1
	}
	if _, err = db.ExecContext(ctx, fmt.Sprintf("PRAGMA busy_timeout = %d", milliseconds)); err != nil {
		return nil, fmt.Errorf("set SQLite busy timeout: %w", err)
	}
	var journalMode string
	if err = db.QueryRowContext(ctx, "PRAGMA journal_mode = WAL").Scan(&journalMode); err != nil {
		return nil, fmt.Errorf("enable SQLite WAL mode: %w", err)
	}
	if !strings.EqualFold(journalMode, "wal") {
		return nil, fmt.Errorf("enable SQLite WAL mode: got %q", journalMode)
	}
	for _, statement := range []string{
		"PRAGMA synchronous = FULL",
		"CREATE TABLE IF NOT EXISTS links (id TEXT PRIMARY KEY NOT NULL, url TEXT NOT NULL)",
	} {
		if _, err = db.ExecContext(ctx, statement); err != nil {
			return nil, fmt.Errorf("initialize SQLite database: %w", err)
		}
	}
	if err = os.Chmod(path, 0600); err != nil {
		return nil, fmt.Errorf("secure SQLite database: %w", err)
	}
	return store, nil
}

func backend(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	if errors.Is(err, sql.ErrNoRows) {
		return links.ErrNotFound
	}
	return fmt.Errorf("%w: %v", links.ErrBackend, err)
}

func (s *Store) Ping(ctx context.Context) error {
	var value int
	return backend(s.db.QueryRowContext(ctx, "SELECT 1").Scan(&value))
}

func (s *Store) Close() error { return s.db.Close() }

func (s *Store) Get(ctx context.Context, id string) (string, error) {
	var target string
	err := s.db.QueryRowContext(ctx, "SELECT url FROM links WHERE id = ?", id).Scan(&target)
	return target, backend(err)
}

func (s *Store) Create(ctx context.Context, id, target string) error {
	result, err := s.db.ExecContext(ctx, "INSERT OR IGNORE INTO links (id, url) VALUES (?, ?)", id, target)
	if err != nil {
		return backend(err)
	}
	created, err := result.RowsAffected()
	if err != nil {
		return backend(err)
	}
	if created == 0 {
		return links.ErrConflict
	}
	return nil
}

func (s *Store) Delete(ctx context.Context, id string) (bool, error) {
	result, err := s.db.ExecContext(ctx, "DELETE FROM links WHERE id = ?", id)
	if err != nil {
		return false, backend(err)
	}
	deleted, err := result.RowsAffected()
	return deleted > 0, backend(err)
}

func (s *Store) List(ctx context.Context) (map[string]string, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT id, url FROM links")
	if err != nil {
		return nil, backend(err)
	}
	defer rows.Close()
	result := make(map[string]string)
	for rows.Next() {
		var id, target string
		if err := rows.Scan(&id, &target); err != nil {
			return nil, backend(err)
		}
		result[id] = target
	}
	if err := rows.Err(); err != nil {
		return nil, backend(err)
	}
	return result, nil
}
