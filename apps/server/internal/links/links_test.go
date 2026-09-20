package links

import (
	"context"
	"errors"
	"testing"
)

type memoryRepo map[string]string

func (m memoryRepo) List(context.Context) (map[string]string, error) { return m, nil }
func (m memoryRepo) Get(_ context.Context, id string) (string, error) {
	v, ok := m[id]
	if !ok {
		return "", ErrNotFound
	}
	return v, nil
}
func (m memoryRepo) Create(_ context.Context, id, target string) error {
	if _, ok := m[id]; ok {
		return ErrConflict
	}
	m[id] = target
	return nil
}
func (m memoryRepo) Delete(_ context.Context, id string) (bool, error) {
	_, ok := m[id]
	delete(m, id)
	return ok, nil
}
func TestServiceContract(t *testing.T) {
	ctx := context.Background()
	repo := memoryRepo{"existing": "https://old.example"}
	s := New(repo)
	calls := 0
	s.newID = func() (string, error) {
		calls++
		if calls == 1 {
			return "existing", nil
		}
		return "new-link", nil
	}
	id, err := s.Create(ctx, "https://example.com/path?q=1")
	if err != nil || id != "new-link" || repo["existing"] != "https://old.example" {
		t.Fatalf("collision handling: %q %v %v", id, err, repo)
	}
	if got, err := s.Get(ctx, id); err != nil || got != repo[id] {
		t.Fatalf("get: %q %v", got, err)
	}
	if data, err := s.List(ctx); err != nil || len(data) != 2 {
		t.Fatalf("list: %v %v", data, err)
	}
	if err := s.Delete(ctx, id); err != nil {
		t.Fatal(err)
	}
	if err := s.Delete(ctx, id); !errors.Is(err, ErrNotFound) {
		t.Fatal(err)
	}
	s.newID = func() (string, error) { return "existing", nil }
	if _, err := s.Create(ctx, "https://example.com"); !errors.Is(err, ErrBackend) {
		t.Fatal(err)
	}
	canceled, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := s.Create(canceled, "https://example.com"); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}
func TestURLValidationAndIDs(t *testing.T) {
	for _, raw := range []string{"", "/relative", "ftp://example.com", "javascript:alert(1)", "http://", "https://example.com/\n", "https://exa mple.com"} {
		if ValidateURL(raw) == nil {
			t.Errorf("accepted %q", raw)
		}
	}
	for _, raw := range []string{"https://example.com", "http://localhost:8080/a?b=c", "https://[::1]/"} {
		if err := ValidateURL(raw); err != nil {
			t.Errorf("%q: %v", raw, err)
		}
	}
	for i := 0; i < 100; i++ {
		id, err := randomID()
		if err != nil || len(id) != 8 {
			t.Fatalf("%q %v", id, err)
		}
	}
}
