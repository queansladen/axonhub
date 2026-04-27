package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/axonhub/axonhub/internal/config"
	"github.com/axonhub/axonhub/internal/server"
)

// main is the entry point for the AxonHub server.
// It initializes configuration, sets up the HTTP server, and handles
// graceful shutdown on SIGINT/SIGTERM signals.
func main() {
	// Load application configuration from environment / config file
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load configuration: %v", err)
	}

	// Build the root HTTP handler (GraphQL, REST, health endpoints)
	handler, err := server.NewHandler(cfg)
	if err != nil {
		log.Fatalf("failed to initialise server handler: %v", err)
	}

	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	httpServer := &http.Server{
		Addr:         addr,
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start listening in a goroutine so we can block on the signal channel
	go func() {
		log.Printf("AxonHub server listening on %s", addr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	// Block until we receive an OS interrupt or termination signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("shutting down server — draining in-flight requests (30s timeout)…")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(ctx); err != nil {
		log.Fatalf("graceful shutdown failed: %v", err)
	}

	log.Println("server stopped cleanly")
}
