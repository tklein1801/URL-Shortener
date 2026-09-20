package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
	"url-shortener/internal/auth"
)

func TestServerLifecycle(t *testing.T) {
	if testing.Short() || runtime.GOOS == "windows" {
		t.Skip("builds server and tests Unix termination signals")
	}
	dir := t.TempDir()
	binary := filepath.Join(dir, "server")
	if output, err := exec.Command("go", "build", "-o", binary, ".").CombinedOutput(); err != nil {
		t.Fatalf("build: %v\n%s", err, output)
	}
	tokenPath := filepath.Join(dir, "data", "master-token")
	databasePath := filepath.Join(dir, "data", "links.db")
	var token string
	var linkID string
	for iteration := 0; iteration < 2; iteration++ {
		listener, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			t.Fatal(err)
		}
		port := listener.Addr().(*net.TCPAddr).Port
		listener.Close()
		outputPath := filepath.Join(dir, fmt.Sprintf("output-%d", iteration))
		out, err := os.Create(outputPath)
		if err != nil {
			t.Fatal(err)
		}
		cmd := exec.Command(binary)
		cmd.Dir = dir
		cmd.Stdout = out
		cmd.Stderr = out
		cmd.Env = append(os.Environ(), "SQLITE_PATH="+databasePath, fmt.Sprintf("PORT=%d", port), "MASTER_TOKEN_FILE="+tokenPath, "SHUTDOWN_TIMEOUT=1s", "BACKEND_TIMEOUT=1s")
		if err := cmd.Start(); err != nil {
			out.Close()
			t.Fatal(err)
		}
		done := make(chan error, 1)
		go func() { done <- cmd.Wait() }()
		stopped := false
		// Cleanup also runs if a readiness assertion fails.
		t.Cleanup(func() {
			if !stopped {
				_ = cmd.Process.Kill()
				<-done
			}
			out.Close()
		})
		client := &http.Client{Timeout: time.Second}
		baseURL := fmt.Sprintf("http://127.0.0.1:%d", port)
		deadline := time.Now().Add(5 * time.Second)
		for {
			response, err := client.Get(baseURL + "/ready")
			if err == nil {
				response.Body.Close()
				if response.StatusCode == 200 {
					break
				}
			}
			if time.Now().After(deadline) {
				t.Fatal("server did not become ready")
			}
			time.Sleep(20 * time.Millisecond)
		}
		current, created, err := auth.Resolve(tokenPath)
		if err != nil || created {
			t.Fatal("token not persisted", err)
		}
		if iteration == 0 {
			token = current
			request, err := http.NewRequest(http.MethodPost, baseURL+"/api/v1/urls", bytes.NewBufferString(`{"url":"https://persisted.example"}`))
			if err != nil {
				t.Fatal(err)
			}
			request.Header.Set("Authorization", "Bearer "+token)
			response, err := client.Do(request)
			if err != nil {
				t.Fatal(err)
			}
			var link struct {
				ID string `json:"id"`
			}
			decodeErr := json.NewDecoder(response.Body).Decode(&link)
			response.Body.Close()
			if response.StatusCode != http.StatusCreated || decodeErr != nil || link.ID == "" {
				t.Fatalf("create persisted link: %d %v", response.StatusCode, decodeErr)
			}
			linkID = link.ID
		} else {
			request, err := http.NewRequest(http.MethodGet, baseURL+"/api/v1/urls", nil)
			if err != nil {
				t.Fatal(err)
			}
			request.Header.Set("Authorization", "Bearer "+token)
			response, err := client.Do(request)
			if err != nil {
				t.Fatal(err)
			}
			var list struct {
				Data map[string]string `json:"data"`
			}
			decodeErr := json.NewDecoder(response.Body).Decode(&list)
			response.Body.Close()
			if response.StatusCode != http.StatusOK || decodeErr != nil || list.Data[linkID] != "https://persisted.example" {
				t.Fatalf("persisted link missing after restart: %d %v %v", response.StatusCode, decodeErr, list.Data)
			}
		}
		if err := cmd.Process.Signal(os.Interrupt); err != nil {
			t.Fatal(err)
		}
		select {
		case err := <-done:
			stopped = true
			if err != nil {
				t.Fatal("shutdown:", err)
			}
		case <-time.After(5 * time.Second):
			t.Fatal("server did not shut down")
		}
		out.Close()
		output, err := os.ReadFile(outputPath)
		if err != nil {
			t.Fatal(err)
		}
		if iteration == 0 {
			if strings.Count(string(output), token) != 1 || !strings.Contains(string(output), "WARNING") {
				t.Fatal("initial token output missing or repeated")
			}
		} else {
			if current != token || strings.Contains(string(output), token) || strings.Contains(string(output), "A new master token") {
				t.Fatal("restart changed or printed token")
			}
		}
	}
}
