// Package remote defines the JSON protocol for DevForge remote
// execution between the CLI client and the DevForge Agent.
package remote

import "time"

// Request is the JSON payload sent from the CLI to the agent.
type Request struct {
	Command     string                 `json:"command"`
	ProjectName string                 `json:"projectName"`
	Config      map[string]interface{} `json:"config,omitempty"`
	Version     string                 `json:"version"`
	Token       string                 `json:"token"`
	DryRun      bool                   `json:"dryRun"`
}

// Response is the JSON payload returned by the agent.
type Response struct {
	Success   bool       `json:"success"`
	Message   string     `json:"message"`
	Logs      []LogEntry `json:"logs,omitempty"`
	Error     string     `json:"error,omitempty"`
	Duration  string     `json:"duration,omitempty"`
	RequestID string     `json:"requestId"`
}

// LogEntry is a single structured log line in the streaming response.
type LogEntry struct {
	Timestamp string `json:"timestamp"`
	Level     string `json:"level"`
	Message   string `json:"message"`
	Module    string `json:"module,omitempty"`
}

// NewLogEntry creates a log entry with the current timestamp.
func NewLogEntry(level, message, module string) LogEntry {
	return LogEntry{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Level:     level,
		Message:   message,
		Module:    module,
	}
}
