package config

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestPaths(t *testing.T) {
	home := t.TempDir()
	xdg := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", xdg)
	if p, err := Path(""); err != nil || p != filepath.Join(xdg, "url-shortener", "config.yaml") {
		t.Fatalf("%s %v", p, err)
	}
	t.Setenv("XDG_CONFIG_HOME", "")
	if p, err := Path(""); err != nil || p != filepath.Join(home, ".config", "url-shortener", "config.yaml") {
		t.Fatalf("%s %v", p, err)
	}
	if p, err := Path("custom.yaml"); err != nil || p != "custom.yaml" {
		t.Fatal(p, err)
	}
	t.Setenv("XDG_CONFIG_HOME", "relative")
	if _, err := Path(""); err == nil {
		t.Fatal("relative XDG accepted")
	}
}
func TestLoadSave(t *testing.T) {
	p := filepath.Join(t.TempDir(), "private", "config.yaml")
	c, err := Load(p)
	if err != nil || c.ServerURL != "http://localhost:3000" {
		t.Fatal(c, err)
	}
	if _, err := os.Stat(filepath.Dir(p)); !os.IsNotExist(err) {
		t.Fatal("load created directory")
	}
	c.Token = "secret"
	if err := Save(p, c); err != nil {
		t.Fatal(err)
	}
	got, err := Load(p)
	if err != nil || got != c {
		t.Fatal(got, err)
	}
	if runtime.GOOS != "windows" {
		for path, mode := range map[string]os.FileMode{p: 0600, filepath.Dir(p): 0700} {
			info, err := os.Stat(path)
			if err != nil {
				t.Fatal(err)
			}
			if info.Mode().Perm() != mode {
				t.Errorf("permissions %o", info.Mode().Perm())
			}
		}
	}
	old, err := os.Open(p)
	if err != nil {
		t.Fatal(err)
	}
	defer old.Close()
	c.Token = "replacement"
	if err := Save(p, c); err != nil {
		t.Fatal(err)
	}
	b := make([]byte, 1024)
	n, _ := old.Read(b)
	if !strings.Contains(string(b[:n]), "secret") {
		t.Fatal("write changed original inode")
	}
	files, err := os.ReadDir(filepath.Dir(p))
	if err != nil || len(files) != 1 {
		t.Fatal("temporary files leaked", files, err)
	}
}
func TestInvalidConfig(t *testing.T) {
	for _, body := range []string{"server_url: /relative", "server_url: 'https://user:secret@example.com'", "token: [secret]", "token: one\ntoken: two", "unknown: secret", "server_url: http://example.com\ntoken: 'a b'"} {
		p := filepath.Join(t.TempDir(), "config.yaml")
		os.WriteFile(p, []byte(body), 0600)
		if _, err := Load(p); err == nil || strings.Contains(err.Error(), "secret") {
			t.Fatalf("unsafe validation: %v", err)
		}
	}
	for _, raw := range []string{"ftp://example.com", "https://example.com?token=secret", "https://example.com/#fragment", "https://example.com?", "https://"} {
		if _, err := ParseURL(raw); err == nil {
			t.Fatalf("accepted %q", raw)
		}
	}
	p := filepath.Join(t.TempDir(), "file")
	os.WriteFile(p, []byte("x"), 0600)
	if err := Save(filepath.Join(p, "config"), Config{ServerURL: "https://example.com"}); err == nil {
		t.Fatal("accepted invalid parent")
	}
}
