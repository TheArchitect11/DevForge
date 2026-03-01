// Package logger provides structured logging for DevForge using logrus.
// It supports console and file output, with configurable verbosity levels
// and optional JSON formatting.
package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/sirupsen/logrus"
)

// Logger wraps logrus to provide structured, leveled logging with
// both console and file output.
type Logger struct {
	entry  *logrus.Entry
	file   *os.File
	mu     sync.Mutex
	closed bool
}

// logDir returns the path to the DevForge log directory.
func logDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(os.TempDir(), ".devforge", "logs")
	}
	return filepath.Join(home, ".devforge", "logs")
}

// logFilePath returns the full path to the DevForge log file.
func logFilePath() string {
	return filepath.Join(logDir(), "devforge.log")
}

// New creates a new Logger instance. If verbose is true, the log level is
// set to Debug; otherwise it defaults to Info. If jsonLogs is true, output
// is formatted as structured JSON. Logs are written to both stderr and a
// persistent log file at ~/.devforge/logs/devforge.log.
func New(verbose bool, jsonLogs ...bool) (*Logger, error) {
	useJSON := len(jsonLogs) > 0 && jsonLogs[0]

	dir := logDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create log directory %s: %w", dir, err)
	}

	fp := logFilePath()
	file, err := os.OpenFile(fp, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file %s: %w", fp, err)
	}

	log := logrus.New()

	// Set console formatter based on JSON flag.
	if useJSON {
		log.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat: "2006-01-02T15:04:05.000Z07:00",
			FieldMap: logrus.FieldMap{
				logrus.FieldKeyTime:  "timestamp",
				logrus.FieldKeyLevel: "level",
				logrus.FieldKeyMsg:   "message",
			},
		})
	} else {
		log.SetFormatter(&logrus.TextFormatter{
			FullTimestamp:   true,
			TimestampFormat: "2006-01-02 15:04:05",
			DisableColors:   false,
		})
	}

	log.SetOutput(os.Stderr)

	if verbose {
		log.SetLevel(logrus.DebugLevel)
	} else {
		log.SetLevel(logrus.InfoLevel)
	}

	// File hook always writes in JSON format for structured log analysis.
	log.AddHook(&fileHook{
		file: file,
		formatter: &logrus.JSONFormatter{
			TimestampFormat: "2006-01-02T15:04:05.000Z07:00",
		},
	})

	entry := logrus.NewEntry(log)

	return &Logger{entry: entry, file: file}, nil
}

// Info logs an informational message.
func (l *Logger) Info(msg string, fields ...map[string]interface{}) {
	l.withFields(fields...).Info(msg)
}

// Warn logs a warning message.
func (l *Logger) Warn(msg string, fields ...map[string]interface{}) {
	l.withFields(fields...).Warn(msg)
}

// Error logs an error message.
func (l *Logger) Error(msg string, fields ...map[string]interface{}) {
	l.withFields(fields...).Error(msg)
}

// Debug logs a debug message (only visible when verbose mode is enabled).
func (l *Logger) Debug(msg string, fields ...map[string]interface{}) {
	l.withFields(fields...).Debug(msg)
}

// Close flushes and closes the underlying log file.
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

// withFields merges optional field maps into the log entry.
func (l *Logger) withFields(fields ...map[string]interface{}) *logrus.Entry {
	if len(fields) == 0 {
		return l.entry
	}
	merged := logrus.Fields{}
	for _, f := range fields {
		for k, v := range f {
			merged[k] = v
		}
	}
	return l.entry.WithFields(merged)
}

// fileHook is a logrus hook that writes every log entry to a file.
type fileHook struct {
	file      *os.File
	formatter logrus.Formatter
}

// Levels returns all log levels so the hook fires on every entry.
func (h *fileHook) Levels() []logrus.Level {
	return logrus.AllLevels
}

// Fire writes the log entry to the file.
func (h *fileHook) Fire(entry *logrus.Entry) error {
	line, err := h.formatter.Format(entry)
	if err != nil {
		return fmt.Errorf("failed to format log entry: %w", err)
	}
	_, err = h.file.Write(line)
	if err != nil {
		return fmt.Errorf("failed to write log entry to file: %w", err)
	}
	return nil
}
