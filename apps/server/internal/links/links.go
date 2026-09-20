// Package links contains application logic independent of HTTP and persistence.
package links

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"net/url"
	"strings"
)

var (
	ErrNotFound   = errors.New("link not found")
	ErrConflict   = errors.New("link already exists")
	ErrBackend    = errors.New("storage unavailable")
	ErrInvalidURL = errors.New("url must be an absolute http or https URL")
)

type Repository interface {
	List(context.Context) (map[string]string, error)
	Get(context.Context, string) (string, error)
	Create(context.Context, string, string) error
	Delete(context.Context, string) (bool, error)
}

type Service struct {
	repo  Repository
	newID func() (string, error)
}

func New(repo Repository) *Service { return &Service{repo: repo, newID: randomID} }
func randomID() (string, error) {
	b := make([]byte, 6)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
func ValidateURL(target string) error {
	u, err := url.Parse(target)
	if err != nil || u == nil || (u.Scheme != "http" && u.Scheme != "https") || u.Hostname() == "" || strings.ContainsAny(target, "\r\n\t ") {
		return ErrInvalidURL
	}
	return nil
}
func (s *Service) Create(ctx context.Context, target string) (string, error) {
	if err := ValidateURL(target); err != nil {
		return "", err
	}
	for i := 0; i < 16; i++ {
		if err := ctx.Err(); err != nil {
			return "", err
		}
		id, err := s.newID()
		if err != nil {
			return "", err
		}
		err = s.repo.Create(ctx, id, target)
		if errors.Is(err, ErrConflict) {
			continue
		}
		return id, err
	}
	return "", ErrBackend
}
func (s *Service) List(ctx context.Context) (map[string]string, error) { return s.repo.List(ctx) }
func (s *Service) Get(ctx context.Context, id string) (string, error)  { return s.repo.Get(ctx, id) }
func (s *Service) Delete(ctx context.Context, id string) error {
	ok, err := s.repo.Delete(ctx, id)
	if err != nil {
		return err
	}
	if !ok {
		return ErrNotFound
	}
	return nil
}
