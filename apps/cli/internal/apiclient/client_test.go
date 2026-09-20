package apiclient

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestClientContract(t *testing.T) {
	var paths []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		if strings.Contains(r.URL.Path, "/api/") && r.Header.Get("Authorization") != "Bearer secret" {
			t.Error("missing bearer token")
		}
		if r.URL.RawQuery != "" {
			t.Error("unexpected query")
		}
		w.Header().Set("Content-Type", "application/json")
		switch r.Method + " " + r.URL.Path {
		case "GET /prefix/health", "GET /prefix/ready":
			w.Write([]byte(`{"status":"ok"}`))
		case "GET /prefix/api/v1/urls":
			w.Write([]byte(`{"data":{"abcdefgh":"https://example.com"}}`))
		case "POST /prefix/api/v1/urls":
			if r.Header.Get("Content-Type") != "application/json" {
				t.Error("wrong content type")
			}
			w.WriteHeader(201)
			w.Write([]byte(`{"id":"abcdefgh","url":"https://example.com"}`))
		case "DELETE /prefix/api/v1/urls/abcdefgh":
			w.Write([]byte(`{"id":"abcdefgh"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	c, err := New(server.URL+"/prefix/", "secret", server.Client())
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if err := c.Status(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := c.Shorten(ctx, "https://example.com"); err != nil {
		t.Fatal(err)
	}
	if _, err := c.Delete(ctx, "abcdefgh"); err != nil {
		t.Fatal(err)
	}
	if target, err := c.RedirectURL("abcdefgh"); err != nil || target != server.URL+"/prefix/r/abcdefgh" {
		t.Fatal(target, err)
	}
	if len(paths) != 5 {
		t.Fatal(paths)
	}
	for _, id := range []string{"..", "a/b", "a?token=secret", "a#b"} {
		if _, err := c.RedirectURL(id); err == nil {
			t.Fatal("unsafe ID", id)
		}
	}
}
func TestFailuresAndRedaction(t *testing.T) {
	for _, status := range []int{400, 401, 404, 503} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(status)
				w.Write([]byte(`{"error":{"message":"secret"}}`))
			}))
			defer server.Close()
			c, _ := New(server.URL, "secret", server.Client())
			_, err := c.List(context.Background())
			var apiErr *APIError
			if !errors.As(err, &apiErr) || apiErr.Status != status || strings.Contains(err.Error(), "secret") {
				t.Fatal(err)
			}
		})
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("invalid secret")) }))
	defer server.Close()
	c, _ := New(server.URL, "secret", server.Client())
	if _, err := c.List(context.Background()); err == nil || strings.Contains(err.Error(), "secret") {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := c.List(ctx); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	c, _ = New(server.URL, "", server.Client())
	if _, err := c.List(context.Background()); err == nil {
		t.Fatal("empty token accepted")
	}
}
func TestRedirectAndTimeout(t *testing.T) {
	calls := 0
	destination := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { calls++ }))
	defer destination.Close()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, destination.URL, 307) }))
	defer server.Close()
	c, _ := New(server.URL, "secret", server.Client())
	if _, err := c.List(context.Background()); err == nil || calls != 0 {
		t.Fatal("redirect followed", err, calls)
	}
	slow := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { <-r.Context().Done() }))
	defer slow.Close()
	c, _ = New(slow.URL, "secret", &http.Client{Timeout: 20 * time.Millisecond})
	if _, err := c.List(context.Background()); err == nil {
		t.Fatal("timeout missing")
	}
}
