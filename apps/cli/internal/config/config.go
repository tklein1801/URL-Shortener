package config

import (
	"errors"
	"gopkg.in/yaml.v2"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

type Config struct {
	ServerURL string `yaml:"server_url" json:"server_url"`
	Token     string `yaml:"token" json:"token"`
}

func Path(custom string) (string, error) {
	if custom != "" {
		return custom, nil
	}
	base := os.Getenv("XDG_CONFIG_HOME")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", errors.New("cannot resolve home directory")
		}
		base = filepath.Join(home, ".config")
	}
	if !filepath.IsAbs(base) {
		return "", errors.New("XDG_CONFIG_HOME must be absolute")
	}
	return filepath.Join(base, "url-shortener", "config.yaml"), nil
}
func ParseURL(raw string) (*url.URL, error) {
	u, err := url.Parse(raw)
	if err != nil || u == nil || (u.Scheme != "http" && u.Scheme != "https") || u.Hostname() == "" || u.User != nil || u.RawQuery != "" || u.ForceQuery || u.Fragment != "" || strings.ContainsAny(raw, "\r\n\t ") {
		return nil, errors.New("server_url must be an absolute http or https URL without credentials, query, or fragment")
	}
	return u, nil
}
func ValidateToken(token string) error {
	if token == "" || strings.ContainsAny(token, " \t\r\n") {
		return errors.New("token must be nonempty and contain no whitespace")
	}
	for _, r := range token {
		if r < 33 || r > 126 {
			return errors.New("token must contain printable ASCII characters")
		}
	}
	return nil
}
func Load(path string) (Config, error) {
	c := Config{ServerURL: "http://localhost:3000"}
	b, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return c, nil
	}
	if err != nil {
		return c, errors.New("cannot read configuration")
	}
	if err := yaml.UnmarshalStrict(b, &c); err != nil {
		return Config{}, errors.New("invalid YAML configuration")
	}
	if _, err := ParseURL(c.ServerURL); err != nil {
		return Config{}, err
	}
	if c.Token != "" {
		if err := ValidateToken(c.Token); err != nil {
			return Config{}, err
		}
	}
	return c, nil
}
func Save(path string, c Config) error {
	if _, err := ParseURL(c.ServerURL); err != nil {
		return err
	}
	if c.Token != "" {
		if err := ValidateToken(c.Token); err != nil {
			return err
		}
	}
	b, err := yaml.Marshal(c)
	if err != nil {
		return errors.New("cannot encode configuration")
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return errors.New("cannot create configuration directory")
	}
	if err := os.Chmod(dir, 0700); err != nil {
		return errors.New("cannot secure configuration directory")
	}
	f, err := os.CreateTemp(dir, ".config-*")
	if err != nil {
		return errors.New("cannot create temporary configuration")
	}
	defer os.Remove(f.Name())
	if err = f.Chmod(0600); err == nil {
		_, err = f.Write(b)
	}
	if err == nil {
		err = f.Sync()
	}
	closeErr := f.Close()
	if err == nil {
		err = closeErr
	}
	if err != nil {
		return errors.New("cannot write configuration")
	}
	if err := os.Rename(f.Name(), path); err != nil {
		return errors.New("cannot replace configuration")
	}
	return nil
}
