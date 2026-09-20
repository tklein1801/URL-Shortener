// Package redisstore adapts Redis string keys to the application repository.
package redisstore

import (
	"context"
	"errors"
	"fmt"
	"github.com/redis/go-redis/v9"
	"url-shortener/internal/links"
)

type Store struct{ client *redis.Client }

var _ links.Repository = (*Store)(nil)

func New(addr, password string, db int) *Store {
	return &Store{client: redis.NewClient(&redis.Options{Addr: addr, Password: password, DB: db, ContextTimeoutEnabled: true})}
}
func backend(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, redis.Nil) {
		return links.ErrNotFound
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	return fmt.Errorf("%w: %v", links.ErrBackend, err)
}
func (s *Store) Ping(ctx context.Context) error { return backend(s.client.Ping(ctx).Err()) }
func (s *Store) Close() error                   { return s.client.Close() }
func (s *Store) Get(ctx context.Context, id string) (string, error) {
	value, err := s.client.Get(ctx, id).Result()
	return value, backend(err)
}
func (s *Store) Create(ctx context.Context, id, target string) error {
	ok, err := s.client.SetNX(ctx, id, target, 0).Result()
	if err != nil {
		return backend(err)
	}
	if !ok {
		return links.ErrConflict
	}
	return nil
}
func (s *Store) Delete(ctx context.Context, id string) (bool, error) {
	n, err := s.client.Del(ctx, id).Result()
	return n > 0, backend(err)
}
func (s *Store) List(ctx context.Context) (map[string]string, error) {
	result := make(map[string]string)
	var cursor uint64
	for {
		keys, next, err := s.client.Scan(ctx, cursor, "*", 100).Result()
		if err != nil {
			return nil, backend(err)
		}
		for _, key := range keys {
			value, err := s.Get(ctx, key)
			// Keys may disappear between SCAN and GET.
			if errors.Is(err, links.ErrNotFound) {
				continue
			}
			if err != nil {
				return nil, err
			}
			result[key] = value
		}
		cursor = next
		if cursor == 0 {
			break
		}
	}
	return result, nil
}
