package remote

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/chinmay/devforge/internal/logger"
)

// Client sends remote execution requests to a DevForge Agent.
type Client struct {
	agentURL   string
	token      string
	httpClient *http.Client
	log        *logger.Logger
}

// NewClient creates a remote execution client.
func NewClient(agentURL, token string, log *logger.Logger, skipTLSVerify bool) *Client {
	transport := &http.Transport{}
	if skipTLSVerify {
		transport.TLSClientConfig = &tls.Config{
			InsecureSkipVerify: true, //nolint:gosec // dev-mode only
		}
	}

	return &Client{
		agentURL: agentURL,
		token:    token,
		httpClient: &http.Client{
			Timeout:   60 * time.Second,
			Transport: transport,
		},
		log: log,
	}
}

// Execute sends a provisioning request to the remote agent and
// returns the response with execution logs.
func (c *Client) Execute(req Request) (*Response, error) {
	req.Token = c.token

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/api/v1/execute", c.agentURL)
	c.log.Info(fmt.Sprintf("sending remote request to %s", url))

	httpReq, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.token)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("remote execution request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 10*1024*1024))
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("agent returned HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	var result Response
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse agent response: %w", err)
	}

	return &result, nil
}

// PrintLogs displays the execution logs from a remote response.
func (c *Client) PrintLogs(resp *Response) {
	for _, entry := range resp.Logs {
		prefix := "INFO"
		switch entry.Level {
		case "error":
			prefix = "ERRO"
		case "warn":
			prefix = "WARN"
		case "debug":
			prefix = "DEBU"
		}
		module := ""
		if entry.Module != "" {
			module = fmt.Sprintf("[%s] ", entry.Module)
		}
		fmt.Printf("  %s %s%s%s\n", entry.Timestamp, prefix, module, entry.Message)
	}
}
