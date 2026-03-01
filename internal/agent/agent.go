// Package agent implements the DevForge Agent — an HTTPS server that
// receives and executes remote provisioning requests.
package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"

	"github.com/chinmay/devforge/internal/audit"
	"github.com/chinmay/devforge/internal/logger"
	"github.com/chinmay/devforge/internal/rbac"
	"github.com/chinmay/devforge/internal/remote"
	devtls "github.com/chinmay/devforge/internal/tls"
)

// Config holds agent configuration.
type Config struct {
	Port          int
	AllowedTokens map[string]rbac.UserInfo // token → user info
	TLS           *devtls.CertPair
	DevMode       bool
}

// Agent is the DevForge remote execution server.
type Agent struct {
	config   Config
	log      *logger.Logger
	auditLog *audit.Logger
	executor *remote.Executor
	server   *http.Server
}

// New creates an Agent with the given configuration.
func New(cfg Config, log *logger.Logger, auditLog *audit.Logger) *Agent {
	executor := remote.NewExecutor(log, auditLog)
	return &Agent{
		config:   cfg,
		log:      log,
		auditLog: auditLog,
		executor: executor,
	}
}

// Start launches the HTTPS server and blocks until shutdown.
func (a *Agent) Start(ctx context.Context) error {
	r := chi.NewRouter()
	r.Use(chimw.RequestID)
	r.Use(chimw.RealIP)
	r.Use(chimw.Recoverer)
	r.Use(chimw.Timeout(60 * time.Second))

	// Health check (unauthenticated).
	r.Get("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
	})

	// Authenticated API routes.
	r.Route("/api/v1", func(r chi.Router) {
		r.Use(a.tokenAuthMiddleware)
		r.With(rbac.RequirePermission(rbac.PermInit)).Post("/execute", a.handleExecute)
	})

	addr := fmt.Sprintf(":%d", a.config.Port)
	a.server = &http.Server{
		Addr:         addr,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 120 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown goroutine.
	go func() {
		<-ctx.Done()
		a.log.Info("shutting down agent...")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := a.server.Shutdown(shutdownCtx); err != nil {
			a.log.Error(fmt.Sprintf("agent shutdown error: %v", err))
		}
	}()

	if a.config.TLS != nil {
		tlsCfg, err := devtls.ServerTLSConfig(a.config.TLS)
		if err != nil {
			return fmt.Errorf("failed to configure TLS: %w", err)
		}
		a.server.TLSConfig = tlsCfg
		a.log.Info(fmt.Sprintf("agent listening on https://0.0.0.0%s", addr))
		return a.server.ListenAndServeTLS(a.config.TLS.CertFile, a.config.TLS.KeyFile)
	}

	if !a.config.DevMode {
		return fmt.Errorf("TLS is required in production mode; use --dev for development")
	}

	a.log.Warn("running in development mode without TLS — NOT for production use")
	a.log.Info(fmt.Sprintf("agent listening on http://0.0.0.0%s", addr))
	return a.server.ListenAndServe()
}

// tokenAuthMiddleware validates Bearer tokens and injects user info
// into the request context.
func (a *Agent) tokenAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, `{"error":"authorization header required"}`, http.StatusUnauthorized)
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			http.Error(w, `{"error":"invalid authorization format"}`, http.StatusUnauthorized)
			return
		}

		token := parts[1]
		user, ok := a.config.AllowedTokens[token]
		if !ok {
			if a.auditLog != nil {
				a.auditLog.LogWithIP("unknown", "auth_failed", "", false, "invalid token", r.RemoteAddr)
			}
			http.Error(w, `{"error":"invalid token"}`, http.StatusUnauthorized)
			return
		}

		ctx := rbac.WithUser(r.Context(), user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// handleExecute processes a remote provisioning request.
func (a *Agent) handleExecute(w http.ResponseWriter, r *http.Request) {
	var req remote.Request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	user, _ := rbac.UserFromContext(r.Context())
	a.log.Info(fmt.Sprintf("remote execution from user %q: %s %s", user.Name, req.Command, req.ProjectName))

	resp := a.executor.Execute(req, user.Name)

	w.Header().Set("Content-Type", "application/json")
	if !resp.Success {
		w.WriteHeader(http.StatusInternalServerError)
	}
	json.NewEncoder(w).Encode(resp)
}
