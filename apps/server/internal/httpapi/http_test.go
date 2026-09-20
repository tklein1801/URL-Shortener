package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
	"url-shortener/internal/auth"
	"url-shortener/internal/links"
)

type repository struct {
	data        map[string]string
	err         error
	sawDeadline bool
}

func (m *repository) check(ctx context.Context) error {
	_, m.sawDeadline = ctx.Deadline()
	return m.err
}
func (m *repository) List(ctx context.Context) (map[string]string, error) {
	return m.data, m.check(ctx)
}
func (m *repository) Get(ctx context.Context, id string) (string, error) {
	if err := m.check(ctx); err != nil {
		return "", err
	}
	v, ok := m.data[id]
	if !ok {
		return "", links.ErrNotFound
	}
	return v, nil
}
func (m *repository) Create(ctx context.Context, id, target string) error {
	if err := m.check(ctx); err != nil {
		return err
	}
	if _, ok := m.data[id]; ok {
		return links.ErrConflict
	}
	m.data[id] = target
	return nil
}
func (m *repository) Delete(ctx context.Context, id string) (bool, error) {
	if err := m.check(ctx); err != nil {
		return false, err
	}
	_, ok := m.data[id]
	delete(m.data, id)
	return ok, nil
}
func TestHTTPContract(t *testing.T) {
	repo := &repository{data: map[string]string{"legacy01": "https://legacy.example"}}
	h := New(links.New(repo), auth.NewVerifier("secret"), func(context.Context) error { return repo.err }, time.Second)
	request := func(method, path, body, token string) *httptest.ResponseRecorder {
		t.Helper()
		r := httptest.NewRequest(method, path, strings.NewReader(body))
		if token != "" {
			r.Header.Set("Authorization", token)
		}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		return w
	}
	for _, tc := range []struct {
		method, path, body, token string
		status                    int
	}{
		{"GET", "/health", "", "", 200}, {"GET", "/ready", "", "", 200},
		{"GET", "/api/v1/urls", "", "", 401}, {"POST", "/api/v1/urls", `{"url":"https://example.com"}`, "", 401}, {"DELETE", "/api/v1/urls/legacy01", "", "", 401},
		{"GET", "/api/v1/urls?code=secret", "", "", 401}, {"GET", "/api/v1/urls", "", "Bearer wrong", 401}, {"GET", "/api/v1/urls", "", "Basic secret", 401},
		{"GET", "/api/v1/urls", "", "bearer secret", 200},
		{"GET", "/r/legacy01", "", "", 307}, {"GET", "/r/missing", "", "", 404},
		{"DELETE", "/api/v1/urls/missing", "", "Bearer secret", 404},
		{"POST", "/api/v1/urls", `{"url":"/relative"}`, "Bearer secret", 400},
		{"POST", "/api/v1/urls", `{"url":"https://example.com","extra":1}`, "Bearer secret", 400},
		{"POST", "/api/v1/urls", `{"url":"https://example.com"} {}`, "Bearer secret", 400},
		{"POST", "/api/v1/urls", `{`, "Bearer secret", 400},
		{"GET", "/list?code=secret", "", "", 404}, {"POST", "/shorten", "", "", 404}, {"PATCH", "/health", "", "", 405},
	} {
		t.Run(tc.method+tc.path+tc.token+tc.body, func(t *testing.T) {
			w := request(tc.method, tc.path, tc.body, tc.token)
			if w.Code != tc.status {
				t.Fatalf("status %d: %s", w.Code, w.Body)
			}
			if tc.status >= 400 {
				var e ErrorResponse
				if err := json.Unmarshal(w.Body.Bytes(), &e); err != nil || e.Error.Code == "" {
					t.Fatalf("invalid error: %s", w.Body)
				}
			}
		})
	}
	w := request("POST", "/api/v1/urls", `{"url":"https://example.com/a?b=c"}`, "Bearer secret")
	var link LinkResponse
	if err := json.Unmarshal(w.Body.Bytes(), &link); err != nil || w.Code != 201 || len(link.ID) != 8 {
		t.Fatalf("create %d %s", w.Code, w.Body)
	}
	if !repo.sawDeadline {
		t.Fatal("request deadline not propagated")
	}
	w = request("GET", "/r/"+link.ID, "", "")
	if w.Code != 307 || w.Header().Get("Location") != link.URL {
		t.Fatal(w)
	}
	if w = request("DELETE", "/api/v1/urls/"+link.ID, "", "Bearer secret"); w.Code != 200 {
		t.Fatal(w)
	}
	repo.err = errors.New("sensitive Redis error")
	for _, path := range []string{"/ready", "/r/legacy01", "/api/v1/urls"} {
		w = request(http.MethodGet, path, "", "Bearer secret")
		if w.Code != 503 || strings.Contains(w.Body.String(), "sensitive") {
			t.Fatalf("outage %s %d %s", path, w.Code, w.Body)
		}
	}
	if w = request("GET", "/health", "", ""); w.Code != 200 {
		t.Fatal("liveness depends on storage")
	}
}
