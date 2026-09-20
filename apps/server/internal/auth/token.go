// Package auth manages the durable master token and constant-time verification.
package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Verifier struct{ digest [32]byte }

func NewVerifier(token string) *Verifier { return &Verifier{digest: sha256.Sum256([]byte(token))} }
func (v *Verifier) Verify(token string) bool {
	digest := sha256.Sum256([]byte(token))
	return subtle.ConstantTimeCompare(v.digest[:], digest[:]) == 1
}
func valid(token string) bool {
	b, err := base64.RawURLEncoding.DecodeString(token)
	return err == nil && len(b) == 32 && base64.RawURLEncoding.EncodeToString(b) == token
}
func read(path string) (string, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return "", err
	}
	if !info.Mode().IsRegular() {
		return "", errors.New("master token must be a regular file")
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return "", errors.New("cannot read master token file")
	}
	token := strings.TrimSuffix(string(b), "\n")
	if !valid(token) {
		return "", errors.New("invalid master token file")
	}
	if err := os.Chmod(path, 0600); err != nil {
		return "", err
	}
	return token, nil
}

// Resolve publishes a fully written file using an exclusive hard link. Concurrent
// starters can only see a complete token; an existing file is never replaced.
func Resolve(path string) (token string, created bool, err error) {
	dir := filepath.Dir(path)
	if err = os.MkdirAll(dir, 0700); err != nil {
		return "", false, err
	}
	if err = os.Chmod(dir, 0700); err != nil {
		return "", false, err
	}
	token, err = read(path)
	if err == nil {
		return token, false, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return "", false, err
	}
	b := make([]byte, 32)
	if _, err = rand.Read(b); err != nil {
		return "", false, err
	}
	token = base64.RawURLEncoding.EncodeToString(b)
	f, err := os.CreateTemp(dir, ".master-token-*")
	if err != nil {
		return "", false, err
	}
	defer os.Remove(f.Name())
	if err = f.Chmod(0600); err == nil {
		_, err = f.WriteString(token + "\n")
	}
	if err == nil {
		err = f.Sync()
	}
	closeErr := f.Close()
	if err == nil {
		err = closeErr
	}
	if err != nil {
		return "", false, err
	}
	if err = os.Link(f.Name(), path); err != nil {
		if errors.Is(err, os.ErrExist) {
			token, err = read(path)
			return token, false, err
		}
		return "", false, fmt.Errorf("publish master token: %w", err)
	}
	// Persist the directory entry before announcing the token.
	d, err := os.Open(dir)
	if err != nil {
		return "", false, err
	}
	err = d.Sync()
	closeErr = d.Close()
	if err == nil {
		err = closeErr
	}
	if err != nil {
		return "", false, err
	}
	return token, true, nil
}
