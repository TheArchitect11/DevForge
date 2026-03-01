package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/chinmay/devforge/internal/logger"
)

// RegistrationRequest is sent from the agent to the central server.
type RegistrationRequest struct {
	MachineID string `json:"machineId"`
	Hostname  string `json:"hostname"`
	Port      int    `json:"port"`
	Version   string `json:"version"`
}

// RegistrationResponse is returned by the central server.
type RegistrationResponse struct {
	Token   string `json:"token"`
	AgentID string `json:"agentId"`
}

// HeartbeatRequest is periodically sent to the central server.
type HeartbeatRequest struct {
	AgentID   string `json:"agentId"`
	MachineID string `json:"machineId"`
	Status    string `json:"status"`
	Uptime    string `json:"uptime"`
}

// Registration manages agent registration and heartbeat with the
// central DevForge server.
type Registration struct {
	serverURL string
	machineID string
	agentID   string
	token     string
	port      int
	version   string
	log       *logger.Logger
	startTime time.Time
}

// NewRegistration creates a Registration instance.
func NewRegistration(serverURL, machineID, version string, port int, log *logger.Logger) *Registration {
	return &Registration{
		serverURL: serverURL,
		machineID: machineID,
		port:      port,
		version:   version,
		log:       log,
		startTime: time.Now(),
	}
}

// Register sends a registration request to the central server.
func (reg *Registration) Register(ctx context.Context) error {
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknown"
	}

	reqBody := RegistrationRequest{
		MachineID: reg.machineID,
		Hostname:  hostname,
		Port:      reg.port,
		Version:   reg.version,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal registration request: %w", err)
	}

	url := fmt.Sprintf("%s/api/v1/agents/register", reg.serverURL)
	reg.log.Info(fmt.Sprintf("registering with server at %s", url))

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create registration request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("registration request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned HTTP %d during registration", resp.StatusCode)
	}

	var regResp RegistrationResponse
	if err := json.NewDecoder(resp.Body).Decode(&regResp); err != nil {
		return fmt.Errorf("failed to parse registration response: %w", err)
	}

	reg.agentID = regResp.AgentID
	reg.token = regResp.Token
	reg.log.Info(fmt.Sprintf("registered successfully as agent %s", reg.agentID))

	return nil
}

// StartHeartbeat runs a background goroutine that periodically sends
// heartbeat signals to the central server. It retries on failure with
// exponential backoff.
func (reg *Registration) StartHeartbeat(ctx context.Context, interval time.Duration) {
	go func() {
		retryDelay := 5 * time.Second
		maxRetryDelay := 2 * time.Minute

		for {
			select {
			case <-ctx.Done():
				reg.log.Info("heartbeat stopped (context cancelled)")
				return
			case <-time.After(interval):
				if err := reg.sendHeartbeat(ctx); err != nil {
					reg.log.Warn(fmt.Sprintf("heartbeat failed: %v (retrying in %s)", err, retryDelay))
					// Exponential backoff.
					select {
					case <-ctx.Done():
						return
					case <-time.After(retryDelay):
					}
					retryDelay *= 2
					if retryDelay > maxRetryDelay {
						retryDelay = maxRetryDelay
					}
				} else {
					retryDelay = 5 * time.Second // Reset on success.
				}
			}
		}
	}()
}

// sendHeartbeat sends a single heartbeat to the central server.
func (reg *Registration) sendHeartbeat(ctx context.Context) error {
	hb := HeartbeatRequest{
		AgentID:   reg.agentID,
		MachineID: reg.machineID,
		Status:    "healthy",
		Uptime:    time.Since(reg.startTime).String(),
	}

	body, err := json.Marshal(hb)
	if err != nil {
		return fmt.Errorf("failed to marshal heartbeat: %w", err)
	}

	url := fmt.Sprintf("%s/api/v1/agents/heartbeat", reg.serverURL)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if reg.token != "" {
		httpReq.Header.Set("Authorization", "Bearer "+reg.token)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("heartbeat returned HTTP %d", resp.StatusCode)
	}

	return nil
}
