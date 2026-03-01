// Package audit provides structured audit logging for DevForge.
// Every critical action is logged with user, timestamp, action,
// success/failure, and machine ID for compliance and SIEM integration.
package audit

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Entry represents a single audit log record.
type Entry struct {
	Timestamp string `json:"timestamp"`
	User      string `json:"user"`
	Action    string `json:"action"`
	Resource  string `json:"resource,omitempty"`
	Success   bool   `json:"success"`
	MachineID string `json:"machineId"`
	Detail    string `json:"detail,omitempty"`
	SourceIP  string `json:"sourceIp,omitempty"`
}

// Logger writes structured audit log entries to a JSON-lines file.
type Logger struct {
	file      *os.File
	mu        sync.Mutex
	machineID string
	closed    bool
}

// auditDir returns the path to the DevForge audit log directory.
func auditDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to determine home directory: %w", err)
	}
	return filepath.Join(home, ".devforge", "audit"), nil
}

// New creates a new audit Logger. The machineID should be a unique
// identifier for the host (hostname, UUID, etc.).
func New(machineID string) (*Logger, error) {
	dir, err := auditDir()
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("failed to create audit directory: %w", err)
	}

	fp := filepath.Join(dir, "audit.log")
	file, err := os.OpenFile(fp, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return nil, fmt.Errorf("failed to open audit log: %w", err)
	}

	return &Logger{file: file, machineID: machineID}, nil
}

// Log writes an audit entry. It is safe for concurrent use.
func (l *Logger) Log(user, action, resource string, success bool, detail string) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.closed {
		return fmt.Errorf("audit logger is closed")
	}

	entry := Entry{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		User:      user,
		Action:    action,
		Resource:  resource,
		Success:   success,
		MachineID: l.machineID,
		Detail:    detail,
	}

	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("failed to marshal audit entry: %w", err)
	}

	if _, err := l.file.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("failed to write audit entry: %w", err)
	}

	return nil
}

// LogWithIP writes an audit entry including the source IP.
func (l *Logger) LogWithIP(user, action, resource string, success bool, detail, sourceIP string) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.closed {
		return fmt.Errorf("audit logger is closed")
	}

	entry := Entry{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		User:      user,
		Action:    action,
		Resource:  resource,
		Success:   success,
		MachineID: l.machineID,
		Detail:    detail,
		SourceIP:  sourceIP,
	}

	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("failed to marshal audit entry: %w", err)
	}

	if _, err := l.file.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("failed to write audit entry: %w", err)
	}

	return nil
}

// Close flushes and closes the audit log file.
func (l *Logger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.closed {
		return nil
	}
	l.closed = true

	if l.file != nil {
		return l.file.Close()
	}
	return nil
}

// GetMachineID returns the machine identifier for this logger.
func (l *Logger) GetMachineID() string {
	return l.machineID
}
