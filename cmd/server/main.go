// DevForge Server — central configuration and policy server.
package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"

	"github.com/chinmay/devforge/internal/logger"
	"github.com/chinmay/devforge/internal/server"
)

var version = "dev"

func main() {
	var (
		port    int
		verbose bool
	)

	flag.IntVar(&port, "port", 8080, "port to listen on")
	flag.BoolVar(&verbose, "verbose", false, "enable verbose logging")
	flag.Parse()

	log, err := logger.New(verbose, false)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer log.Close()

	store := server.NewMemoryStorage()

	r := chi.NewRouter()
	r.Use(chimw.RequestID)
	r.Use(chimw.RealIP)
	r.Use(chimw.Recoverer)
	r.Use(chimw.Timeout(30 * time.Second))

	// Health check.
	r.Get("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok","version":"` + version + `"}`))
	})

	// Register all API handlers.
	server.RegisterHandlers(r, store)

	addr := fmt.Sprintf(":%d", port)
	srv := &http.Server{
		Addr:         addr,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigChan
		log.Info(fmt.Sprintf("received signal: %s", sig))
		cancel()
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer shutdownCancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			log.Error(fmt.Sprintf("server shutdown error: %v", err))
		}
	}()

	fmt.Printf("DevForge Server v%s\n", version)
	log.Info(fmt.Sprintf("server listening on http://0.0.0.0%s", addr))

	_ = ctx // used by signal handler goroutine
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Error(fmt.Sprintf("server error: %v", err))
		os.Exit(1)
	}

	log.Info("server stopped gracefully")
}
