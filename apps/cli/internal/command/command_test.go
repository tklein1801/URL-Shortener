package command

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
	"url-shortener-cli/internal/apiclient"
	"url-shortener-cli/internal/config"
)

type fakeAPI struct{ err error }

func (f fakeAPI) Status(context.Context) error { return f.err }
func (f fakeAPI) List(context.Context) (apiclient.ListResponse, error) {
	return apiclient.ListResponse{Data: map[string]string{"bbbbbbbb": "https://b.example", "aaaaaaaa": "https://a.example"}}, f.err
}
func (f fakeAPI) Shorten(context.Context, string) (apiclient.Link, error) {
	return apiclient.Link{ID: "aaaaaaaa", URL: "https://a.example"}, f.err
}
func (f fakeAPI) Delete(context.Context, string) (apiclient.Link, error) {
	return apiclient.Link{ID: "aaaaaaaa"}, f.err
}
func (f fakeAPI) RedirectURL(id string) (string, error) {
	return "https://short.example/r/" + id, f.err
}
func TestHelpDoesNotTouchConfig(t *testing.T) {
	base := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", base)
	for _, args := range [][]string{{"--help"}, {"config", "--help"}, {"version"}} {
		var out, diag bytes.Buffer
		if code := Execute(Dependencies{Out: &out, Err: &diag}, args); code != 0 {
			t.Fatal(code, diag.String())
		}
	}
	entries, _ := os.ReadDir(base)
	if len(entries) != 0 {
		t.Fatal("help/version created configuration")
	}
	p := filepath.Join(base, "bad.yaml")
	os.WriteFile(p, []byte("invalid: ["), 0600)
	var out bytes.Buffer
	if code := Execute(Dependencies{Out: &out, Err: &out}, []string{"--config", p, "--help"}); code != 0 {
		t.Fatal(out.String())
	}
}
func TestCommands(t *testing.T) {
	p := filepath.Join(t.TempDir(), "config.yaml")
	for _, tc := range []struct {
		args []string
		want string
		fail bool
	}{
		{[]string{"config", "set-server", "https://short.example"}, "saved", false},
		{[]string{"config", "set-token"}, "saved", false},
		{[]string{"config", "show"}, "********", false},
		{[]string{"config", "show", "--json"}, `"token":"********"`, false},
		{[]string{"status"}, "authentication valid", false},
		{[]string{"list"}, "aaaaaaaa\thttps://a.example\nbbbbbbbb\thttps://b.example\n", false},
		{[]string{"shorten", "https://a.example", "--json"}, `"short_url":"https://short.example/r/aaaaaaaa"`, false},
		{[]string{"open", "aaaaaaaa"}, "https://short.example/r/aaaaaaaa", false},
		{[]string{"delete", "aaaaaaaa"}, "Deleted aaaaaaaa", false},
		{[]string{"version", "--json"}, `"version":"v1.2.3"`, false},
		{[]string{"delete"}, "", true}, {[]string{"--timeout", "0s", "list"}, "", true},
	} {
		t.Run(strings.Join(tc.args, " "), func(t *testing.T) {
			var out, diag bytes.Buffer
			d := Dependencies{Out: &out, Err: &diag, Version: "v1.2.3", ReadToken: func(io.Reader) (string, error) { return "super-secret-token", nil }, OpenBrowser: func(context.Context, string) error { return nil }, NewClient: func(config.Config, time.Duration) (API, error) { return fakeAPI{}, nil }}
			code := Execute(d, append([]string{"--config", p}, tc.args...))
			if (code != 0) != tc.fail || !strings.Contains(out.String(), tc.want) {
				t.Fatalf("code %d out %q err %q", code, out.String(), diag.String())
			}
			if strings.Contains(out.String()+diag.String(), "super-secret-token") {
				t.Fatal("token leaked")
			}
		})
	}
}
func TestBrowserAndAPIFailures(t *testing.T) {
	for _, browser := range []bool{false, true} {
		var out, diag bytes.Buffer
		d := Dependencies{Out: &out, Err: &diag, OpenBrowser: func(context.Context, string) error { return errors.New("sensitive internal error") }, NewClient: func(config.Config, time.Duration) (API, error) {
			if browser {
				return fakeAPI{}, nil
			}
			return fakeAPI{err: &apiclient.APIError{Status: 404}}, nil
		}}
		args := []string{"--config", filepath.Join(t.TempDir(), "config.yaml"), "delete", "aaaaaaaa"}
		if browser {
			args[2] = "open"
		}
		if code := Execute(d, args); code != 1 {
			t.Fatal(code)
		}
		if browser && (!strings.Contains(out.String(), "https://short.example/r/aaaaaaaa") || !strings.Contains(diag.String(), "browser launch failed")) {
			t.Fatal(out.String(), diag.String())
		}
		if !browser && !strings.Contains(diag.String(), "link not found") {
			t.Fatal(diag.String())
		}
	}
}
func TestReadTokenFromPipe(t *testing.T) {
	for _, input := range []string{"secret\n", "secret\r\n", "secret"} {
		token, err := readToken(strings.NewReader(input))
		if err != nil || token != "secret" {
			t.Fatal(token, err)
		}
	}
	if _, err := readToken(strings.NewReader(strings.Repeat("x", 4097))); err == nil {
		t.Fatal("unbounded input accepted")
	}
}
