package auth

import (
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
)

func TestResolveReusePermissions(t *testing.T) {
	p := filepath.Join(t.TempDir(), "private", "master-token")
	token, created, err := Resolve(p)
	if err != nil || !created || !valid(token) {
		t.Fatalf("create: %v %v", created, err)
	}
	reused, created, err := Resolve(p)
	if err != nil || created || reused != token {
		t.Fatalf("reuse: %v %v", created, err)
	}
	if runtime.GOOS != "windows" {
		for path, mode := range map[string]os.FileMode{p: 0600, filepath.Dir(p): 0700} {
			info, err := os.Stat(path)
			if err != nil {
				t.Fatal(err)
			}
			if info.Mode().Perm() != mode {
				t.Errorf("mode %s = %o", path, info.Mode().Perm())
			}
		}
	}
	verifier := NewVerifier(token)
	if !verifier.Verify(token) || verifier.Verify(token+"x") || verifier.Verify("") {
		t.Fatal("verification failed")
	}
}
func TestInvalidFilesNeverReplaced(t *testing.T) {
	for _, body := range []string{"", "weak-token", "bad\ncontents", "___________________________________________"} {
		t.Run(body, func(t *testing.T) {
			p := filepath.Join(t.TempDir(), "token")
			if err := os.WriteFile(p, []byte(body), 0600); err != nil {
				t.Fatal(err)
			}
			if _, _, err := Resolve(p); err == nil {
				t.Fatal("invalid file accepted")
			}
			got, _ := os.ReadFile(p)
			if string(got) != body {
				t.Fatal("existing token replaced")
			}
		})
	}
	t.Run("directory", func(t *testing.T) {
		p := filepath.Join(t.TempDir(), "token")
		os.Mkdir(p, 0700)
		if _, _, err := Resolve(p); err == nil {
			t.Fatal("accepted directory")
		}
	})
	t.Run("parent-file", func(t *testing.T) {
		p := filepath.Join(t.TempDir(), "file")
		os.WriteFile(p, []byte("x"), 0600)
		if _, _, err := Resolve(filepath.Join(p, "token")); err == nil {
			t.Fatal("accepted file parent")
		}
	})
	t.Run("symlink", func(t *testing.T) {
		dir := t.TempDir()
		target := filepath.Join(dir, "target")
		os.WriteFile(target, []byte("unchanged"), 0600)
		p := filepath.Join(dir, "token")
		if err := os.Symlink(target, p); err != nil {
			t.Skip(err)
		}
		if _, _, err := Resolve(p); err == nil {
			t.Fatal("accepted symlink")
		}
	})
	t.Run("unreadable", func(t *testing.T) {
		if runtime.GOOS == "windows" || os.Geteuid() == 0 {
			t.Skip("requires Unix non-root permissions")
		}
		p := filepath.Join(t.TempDir(), "token")
		Resolve(p)
		os.Chmod(p, 0000)
		defer os.Chmod(p, 0600)
		if _, _, err := Resolve(p); err == nil {
			t.Fatal("accepted unreadable token")
		}
	})
}
func TestConcurrentResolve(t *testing.T) {
	p := filepath.Join(t.TempDir(), "data", "token")
	const n = 24
	tokens := make([]string, n)
	created := make([]bool, n)
	errs := make([]error, n)
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) { defer wg.Done(); tokens[i], created[i], errs[i] = Resolve(p) }(i)
	}
	wg.Wait()
	count := 0
	for i := range tokens {
		if errs[i] != nil {
			t.Fatal(errs[i])
		}
		if tokens[i] != tokens[0] {
			t.Fatal("different tokens")
		}
		if created[i] {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("created %d times", count)
	}
}
