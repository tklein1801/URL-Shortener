package main

import (
	"encoding/json"
	"github.com/alicebob/miniredis/v2"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
	"url-shortener/internal/auth"
	"url-shortener/internal/httpapi"
	"url-shortener/internal/links"
	"url-shortener/internal/redisstore"
)

// Exercise the real CLI binary against the HTTP stack and Redis adapter.
func TestCLIWorkflow(t *testing.T) {
	if testing.Short() {
		t.Skip("builds CLI executable")
	}
	dir := t.TempDir()
	binary := filepath.Join(dir, "surl")
	if runtime.GOOS == "windows" {
		binary += ".exe"
	}
	build := exec.Command("go", "build", "-o", binary, ".")
	build.Dir = "../cli"
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build CLI: %v\n%s", err, output)
	}
	token, _, err := auth.Resolve(filepath.Join(dir, "data", "master-token"))
	if err != nil {
		t.Fatal(err)
	}
	redis := miniredis.RunT(t)
	store := redisstore.New(redis.Addr(), "", 0)
	defer store.Close()
	server := httptest.NewServer(httpapi.New(links.New(store), auth.NewVerifier(token), store.Ping, time.Second))
	defer server.Close()
	config := filepath.Join(dir, "config", "config.yaml")
	run := func(input string, args ...string) string {
		t.Helper()
		cmd := exec.Command(binary, append([]string{"--config", config}, args...)...)
		cmd.Stdin = strings.NewReader(input)
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("CLI %v: %v\n%s", args, err, output)
		}
		if strings.Contains(string(output), token) {
			t.Fatal("token leaked")
		}
		return string(output)
	}
	run("", "config", "set-server", server.URL)
	run(token+"\n", "config", "set-token")
	run("", "status")
	result := run("", "shorten", "https://example.com/path?q=1", "--json")
	var link struct {
		ID       string `json:"id"`
		ShortURL string `json:"short_url"`
	}
	if err := json.Unmarshal([]byte(result), &link); err != nil || len(link.ID) != 8 {
		t.Fatal(result, err)
	}
	if out := run("", "list"); !strings.Contains(out, link.ID) {
		t.Fatal(out)
	}
	client := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	response, err := client.Get(link.ShortURL)
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != 307 || response.Header.Get("Location") != "https://example.com/path?q=1" {
		t.Fatal(response)
	}
	if runtime.GOOS == "linux" {
		// A local stub verifies browser arguments without opening a real browser.
		stub := filepath.Join(dir, "xdg-open")
		if err := os.WriteFile(stub, []byte("#!/bin/sh\n[ \"$1\" = \"$EXPECTED_REDIRECT\" ]\n"), 0700); err != nil {
			t.Fatal(err)
		}
		t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
		t.Setenv("EXPECTED_REDIRECT", link.ShortURL)
		run("", "open", link.ID)
	}
	run("", "delete", link.ID)
	if out := run("", "list", "--json"); !strings.Contains(out, `"data":{}`) {
		t.Fatal(out)
	}
	cmd := exec.Command(binary, "--config", config, "delete", link.ID)
	output, err := cmd.CombinedOutput()
	if err == nil || !strings.Contains(string(output), "link not found") {
		t.Fatal(string(output), err)
	}
}
