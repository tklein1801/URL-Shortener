package config

import (
	"errors"
	"fmt"
	"github.com/joho/godotenv"
	"net"
	"os"
	"strconv"
	"time"
)

type Config struct {
	Addr, RedisAddr, RedisPassword, TokenFile                               string
	RedisDB                                                                 int
	ReadTimeout, WriteTimeout, IdleTimeout, ShutdownTimeout, BackendTimeout time.Duration
}

func Load() (Config, error) {
	if err := godotenv.Load(); err != nil && !errors.Is(err, os.ErrNotExist) {
		return Config{}, errors.New("cannot load .env")
	}
	return Parse(os.Getenv)
}

// Parse allows testing without changing process-global environment variables.
func Parse(get func(string) string) (Config, error) {
	value := func(key, fallback string) string {
		if s := get(key); s != "" {
			return s
		}
		return fallback
	}
	c := Config{RedisAddr: value("REDIS_HOST", "localhost:6379"), RedisPassword: get("REDIS_PW"), TokenFile: value("MASTER_TOKEN_FILE", "./data/master-token")}
	port, err := strconv.Atoi(value("PORT", "3000"))
	if err != nil || port < 1 || port > 65535 {
		return c, errors.New("PORT must be between 1 and 65535")
	}
	c.Addr = net.JoinHostPort("", strconv.Itoa(port))
	if _, _, err := net.SplitHostPort(c.RedisAddr); err != nil {
		return c, errors.New("REDIS_HOST must be host:port")
	}
	c.RedisDB, err = strconv.Atoi(value("REDIS_DB", "0"))
	if err != nil || c.RedisDB < 0 {
		return c, errors.New("REDIS_DB must be a nonnegative integer")
	}
	for _, setting := range []struct {
		key, fallback string
		dest          *time.Duration
	}{
		{"HTTP_READ_TIMEOUT", "10s", &c.ReadTimeout}, {"HTTP_WRITE_TIMEOUT", "15s", &c.WriteTimeout},
		{"HTTP_IDLE_TIMEOUT", "60s", &c.IdleTimeout}, {"SHUTDOWN_TIMEOUT", "10s", &c.ShutdownTimeout},
		{"BACKEND_TIMEOUT", "5s", &c.BackendTimeout},
	} {
		*setting.dest, err = time.ParseDuration(value(setting.key, setting.fallback))
		if err != nil || *setting.dest <= 0 {
			return c, fmt.Errorf("%s must be a positive duration", setting.key)
		}
	}
	return c, nil
}
