package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/lacsar712/stacklift/internal/app"
)

func main() {
	demo := flag.Bool("demo", false, "run demo retrieval cycle")
	flag.Parse()

	cfg, err := app.LoadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "config error: %v\n", err)
		os.Exit(1)
	}
	if *demo {
		cfg.DemoMode = true
	}

	application, err := app.New(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "init error: %v\n", err)
		os.Exit(1)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	if err := application.Start(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "start error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("stacklift ASRS controller started (%d cranes)\n", application.CraneCount())

	if cfg.DemoMode {
		if err := app.RunDemo(ctx, application); err != nil {
			fmt.Fprintf(os.Stderr, "demo error: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("demo cycle complete")
		application.Shutdown(ctx)
		return
	}

	<-ctx.Done()
	fmt.Println("shutting down")
	application.Shutdown(ctx)
}
