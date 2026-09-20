package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/go-chi/chi/v5"
	"io"
	"net/http"
	"strings"
	"time"
	"url-shortener/internal/auth"
	"url-shortener/internal/links"
)

type ErrorResponse struct {
	Error APIError `json:"error"`
}
type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}
type CreateRequest struct {
	URL string `json:"url"`
}
type LinkResponse struct {
	ID  string `json:"id"`
	URL string `json:"url,omitempty"`
}
type ListResponse struct {
	Data map[string]string `json:"data"`
}

func write(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}
func fail(w http.ResponseWriter, status int, code, message string) {
	write(w, status, ErrorResponse{APIError{code, message}})
}
func failure(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, links.ErrNotFound):
		fail(w, 404, "not_found", "link not found")
	case errors.Is(err, links.ErrInvalidURL):
		fail(w, 400, "invalid_url", links.ErrInvalidURL.Error())
	default:
		fail(w, 503, "unavailable", "storage unavailable")
	}
}
func New(service *links.Service, verifier *auth.Verifier, ping func(context.Context) error, timeout time.Duration) http.Handler {
	r := chi.NewRouter()
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, cancel := context.WithTimeout(r.Context(), timeout)
			defer cancel()
			defer func() {
				if recover() != nil {
					fail(w, 500, "internal_error", "internal server error")
				}
			}()
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	})
	r.NotFound(func(w http.ResponseWriter, r *http.Request) { fail(w, 404, "not_found", "route not found") })
	r.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) { fail(w, 405, "method_not_allowed", "method not allowed") })
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) { write(w, 200, map[string]string{"status": "ok"}) })
	r.Get("/ready", func(w http.ResponseWriter, r *http.Request) {
		if err := ping(r.Context()); err != nil {
			failure(w, err)
			return
		}
		write(w, 200, map[string]string{"status": "ok"})
	})
	r.Route("/api/v1/urls", func(r chi.Router) {
		r.Use(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				fields := strings.Fields(r.Header.Get("Authorization"))
				if len(r.Header.Values("Authorization")) != 1 || len(fields) != 2 || !strings.EqualFold(fields[0], "Bearer") || !verifier.Verify(fields[1]) {
					w.Header().Set("WWW-Authenticate", "Bearer")
					fail(w, 401, "unauthorized", "valid bearer token required")
					return
				}
				next.ServeHTTP(w, r)
			})
		})
		r.Get("/", func(w http.ResponseWriter, r *http.Request) {
			data, err := service.List(r.Context())
			if err != nil {
				failure(w, err)
				return
			}
			write(w, 200, ListResponse{data})
		})
		r.Post("/", func(w http.ResponseWriter, r *http.Request) {
			dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
			dec.DisallowUnknownFields()
			var input CreateRequest
			if err := dec.Decode(&input); err != nil {
				fail(w, 400, "invalid_request", "expected JSON object with url")
				return
			}
			if err := dec.Decode(new(any)); err != io.EOF {
				fail(w, 400, "invalid_request", "expected one JSON object")
				return
			}
			id, err := service.Create(r.Context(), input.URL)
			if err != nil {
				failure(w, err)
				return
			}
			w.Header().Set("Location", "/r/"+id)
			write(w, 201, LinkResponse{ID: id, URL: input.URL})
		})
		r.Delete("/{id}", func(w http.ResponseWriter, r *http.Request) {
			id := chi.URLParam(r, "id")
			if err := service.Delete(r.Context(), id); err != nil {
				failure(w, err)
				return
			}
			write(w, 200, LinkResponse{ID: id})
		})
	})
	r.Get("/r/{id}", func(w http.ResponseWriter, r *http.Request) {
		target, err := service.Get(r.Context(), chi.URLParam(r, "id"))
		if err != nil {
			failure(w, err)
			return
		}
		http.Redirect(w, r, target, http.StatusTemporaryRedirect)
	})
	return r
}
