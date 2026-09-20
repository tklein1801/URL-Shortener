// Package apiclient implements the versioned management API.
package apiclient

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"url-shortener-cli/internal/config"
)

type Link struct {
	ID  string `json:"id"`
	URL string `json:"url,omitempty"`
}
type ListResponse struct {
	Data map[string]string `json:"data"`
}
type APIError struct {
	Status int
	Code   string
}

func (e *APIError) Error() string {
	switch e.Status {
	case 401:
		return "authentication failed; configure a valid token"
	case 404:
		return "link not found"
	case 400:
		return "server rejected the request"
	case 503:
		return "server storage unavailable"
	}
	return fmt.Sprintf("server returned HTTP %d", e.Status)
}

type Client struct {
	base  *url.URL
	token string
	http  *http.Client
}

func New(raw, token string, client *http.Client) (*Client, error) {
	base, err := config.ParseURL(raw)
	if err != nil {
		return nil, err
	}
	if token != "" {
		if err := config.ValidateToken(token); err != nil {
			return nil, err
		}
	}
	if client == nil {
		return nil, errors.New("HTTP client is required")
	}
	copyClient := *client
	// Never follow management redirects or forward credentials elsewhere.
	copyClient.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	return &Client{base: base, token: token, http: &copyClient}, nil
}
func validID(id string) bool {
	if id == "" || id == "." || id == ".." {
		return false
	}
	for _, c := range id {
		if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '-' || c == '_') {
			return false
		}
	}
	return true
}
func (c *Client) endpoint(path string) string {
	u := *c.base
	u.Path = strings.TrimRight(u.Path, "/") + path
	u.RawPath = ""
	return u.String()
}
func (c *Client) RedirectURL(id string) (string, error) {
	if !validID(id) {
		return "", errors.New("invalid short ID")
	}
	return c.endpoint("/r/" + id), nil
}
func (c *Client) request(ctx context.Context, method, path string, input, output any, auth bool) error {
	var body io.Reader
	if input != nil {
		b, err := json.Marshal(input)
		if err != nil {
			return errors.New("cannot encode request")
		}
		body = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.endpoint(path), body)
	if err != nil {
		return errors.New("cannot construct request")
	}
	if auth {
		if c.token == "" {
			return errors.New("token is not configured; run surl config set-token")
		}
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	if input != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.http.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		// Transport errors and remote bodies may contain secrets. Do not echo them.
		return errors.New("server request failed (connection, TLS, or timeout)")
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return &APIError{Status: resp.StatusCode}
	}
	if output != nil {
		if err := json.NewDecoder(io.LimitReader(resp.Body, 16<<20)).Decode(output); err != nil {
			return errors.New("invalid server response")
		}
	}
	return nil
}
func (c *Client) Status(ctx context.Context) error {
	if err := c.request(ctx, "GET", "/health", nil, nil, false); err != nil {
		return err
	}
	if err := c.request(ctx, "GET", "/ready", nil, nil, false); err != nil {
		return err
	}
	_, err := c.List(ctx)
	return err
}
func (c *Client) List(ctx context.Context) (ListResponse, error) {
	var out ListResponse
	err := c.request(ctx, "GET", "/api/v1/urls", nil, &out, true)
	return out, err
}
func (c *Client) Shorten(ctx context.Context, target string) (Link, error) {
	var out Link
	err := c.request(ctx, "POST", "/api/v1/urls", map[string]string{"url": target}, &out, true)
	if err == nil && !validID(out.ID) {
		return Link{}, errors.New("invalid server response")
	}
	return out, err
}
func (c *Client) Delete(ctx context.Context, id string) (Link, error) {
	if !validID(id) {
		return Link{}, errors.New("invalid short ID")
	}
	var out Link
	err := c.request(ctx, "DELETE", "/api/v1/urls/"+id, nil, &out, true)
	return out, err
}
