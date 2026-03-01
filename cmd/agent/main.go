// DevForge Agent — HTTPS server for remote provisioning execution.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/chinmay/devforge/internal/agent"
	"github.com/chinmay/devforge/internal/audit"
	"github.com/chinmay/devforge/internal/logger"
	"github.com/chinmay/devforge/internal/rbac"
	devtls "github.com/chinmay/devforge/internal/tls"
)

var version = "dev"

func main() {
	var (
		port       int
		devMode    bool
		verbose    bool
		serverURL  string
		agentToken string
	)

	flag.IntVar(&port, "port", 8443, "port to listen on")
	flag.BoolVar(&devMode, "dev", false, "enable development mode (no TLS)")
	flag.BoolVar(&verbose, "verbose", false, "enable verbose logging")
	flag.StringVar(&serverURL, "server", "", "central server URL for registration")
	flag.StringVar(&agentToken, "token", "", "authentication token for agent access")
	flag.Parse()

	log, err := logger.New(verbose, false)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer log.Close()

	hostname, _ := os.Hostname()
	auditLog, err := audit.New(hostname)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to initialize audit logger: %v\n", err)
		os.Exit(1)
	}
	defer auditLog.Close()

	// Set up allowed tokens. In production, load from config file.
	allowedTokens := map[string]rbac.UserInfo{}
	if agentToken != "" {
		allowedTokens[agentToken] = rbac.UserInfo{
			ID:   "default",
			Name: "agent-user",
			Role: rbac.RoleAdmin,
		}
	}

	cfg := agent.Config{
		Port:          port,
		AllowedTokens: allowedTokens,
		DevMode:       devMode,
	}

	// TLS setup.
	if !devMode {
		certPair, err := devtls.GenerateSelfSigned([]string{"localhost", "127.0.0.1", hostname})
		if err != nil {
			log.Error(fmt.Sprintf("TLS setup failed: %v", err))
			os.Exit(1)
		}
		cfg.TLS = certPair
	}

	ag := agent.New(cfg, log, auditLog)

	// Context with cancellation for graceful shutdown.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle OS signals.
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigChan
		log.Info(fmt.Sprintf("received signal: %s", sig))
		cancel()
	}()

	// Register with central server if configured.
	if serverURL != "" {
		reg := agent.NewRegistration(serverURL, hostname, version, port, log)
		if err := reg.Register(ctx); err != nil {
			log.Warn(fmt.Sprintf("server registration failed: %v (continuing standalone)", err))
		} else {
			reg.StartHeartbeat(ctx, 30*60*1000*1000*1000) // 30 seconds in nanoseconds
		}
	}

	fmt.Printf("DevForge Agent v%s\n", version)
	if err := ag.Start(ctx); err != nil {
		if err.Error() != "http: Server closed" {
			log.Error(fmt.Sprintf("agent error: %v", err))
			os.Exit(1)
		}
	}

	log.Info("agent stopped gracefully")
}
