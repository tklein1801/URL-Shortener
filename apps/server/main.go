package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"url-shortener/internal/auth"
	"url-shortener/internal/config"
	"url-shortener/internal/httpapi"
	"url-shortener/internal/links"
	"url-shortener/internal/redisstore"
)

func run() error {
	c, err := config.Load()
	if err != nil {
		return err
	}
	token, created, err := auth.Resolve(c.TokenFile)
	if err != nil {
		return fmt.Errorf("master token: %w", err)
	}
	if created {
		fmt.Printf("A new master token was generated.\nWARNING: This token grants full management access.\nStore this token securely. It will not be printed again:\n\n%s\n\nToken file: %s\n", token, c.TokenFile)
	}
	store := redisstore.New(c.RedisAddr, c.RedisPassword, c.RedisDB)
	defer store.Close()
	ctx, cancel := context.WithTimeout(context.Background(), c.BackendTimeout)
	err = store.Ping(ctx)
	cancel()
	if err != nil {
		return errors.New("Redis startup check failed")
	}
	server := &http.Server{Addr: c.Addr, Handler: httpapi.New(links.New(store), auth.NewVerifier(token), store.Ping, c.BackendTimeout), ReadHeaderTimeout: c.ReadTimeout, ReadTimeout: c.ReadTimeout, WriteTimeout: c.WriteTimeout, IdleTimeout: c.IdleTimeout}
	signals, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	done := make(chan error, 1)
	go func() { done <- server.ListenAndServe() }()
	log.Printf("HTTP server listening on %s", c.Addr)
	select {
	case err := <-done:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-signals.Done():
		ctx, cancel := context.WithTimeout(context.Background(), c.ShutdownTimeout)
		defer cancel()
		if err := server.Shutdown(ctx); err != nil {
			_ = server.Close()
			return err
		}
		err := <-done
		if !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		return nil
	}
}
func main() {
	if err := run(); err != nil {
		log.Print(err)
		os.Exit(1)
	}
}
