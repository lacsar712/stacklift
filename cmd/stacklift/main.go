package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/lacsar712/stacklift/internal/app"
	"github.com/lacsar712/stacklift/internal/config"
	"github.com/lacsar712/stacklift/internal/web"
)

func main() {
	cfg := config.Default()
	if path := os.Getenv("CRANESAFE_CONFIG"); path != "" {
		loaded, err := config.LoadFile(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "config: %v\n", err)
			os.Exit(1)
		}
		cfg = loaded
	}
	svc, err := app.New(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "app: %v\n", err)
		os.Exit(1)
	}
	mux := http.NewServeMux()
	mux.Handle("/api/", svc.Server.Handler())
	mux.Handle("/", web.Handler())
	httpSrv := &http.Server{Addr: cfg.ListenAddr, Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	errCh := make(chan error, 1)
	go func() {
		fmt.Printf("stacklift listening on %s\n", cfg.ListenAddr)
		errCh <- httpSrv.ListenAndServe()
	}()
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	select {
	case sig := <-sigCh:
		fmt.Printf("shutdown: %s\n", sig)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = httpSrv.Shutdown(ctx)
		svc.Cancel()
	case err := <-errCh:
		if err != nil && err != http.ErrServerClosed {
			fmt.Fprintf(os.Stderr, "http: %v\n", err)
			os.Exit(1)
		}
	}
}
