// Package logger provides structured logging for DevForge using logrus.
// Console output is suppressed unless --verbose is set; the log file at
// ~/.devforge/logs/devforge.log always receives all entries in JSON format.
package logger

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"

	"github.com/sirupsen/logrus"
)

// Logger wraps logrus for structured, levelled, dual-output logging.
type Logger struct {
	entry  *logrus.Entry
	file   *os.File
	mu     sync.Mutex
	closed bool
}

func logDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(os.TempDir(), ".devforge", "logs")
	}
	return filepath.Join(home, ".devforge", "logs")
}

func logFilePath() string { return filepath.Join(logDir(), "devforge.log") }

// New creates a Logger.
//   - verbose=true  → debug level; entries written to both stderr and the log file.
//   - verbose=false → info level;  entries written only to the log file (stderr silent).
//   - jsonLogs=true → stderr output formatted as JSON (useful in CI).
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

	if verbose {
		// Verbose: write to stderr so the user can see debug output.
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
		log.SetLevel(logrus.DebugLevel)
	} else {
		// Non-verbose: suppress console noise; the file hook handles persistence.
		log.SetOutput(io.Discard)
		log.SetLevel(logrus.InfoLevel)
	}

	// The file hook always writes structured JSON for offline analysis.
	log.AddHook(&fileHook{
		file: file,
		formatter: &logrus.JSONFormatter{
			TimestampFormat: "2006-01-02T15:04:05.000Z07:00",
		},
	})

	return &Logger{entry: logrus.NewEntry(log), file: file}, nil
}

func (l *Logger) Info(msg string, fields ...map[string]interface{}) {
	l.withFields(fields...).Info(msg)
}
func (l *Logger) Warn(msg string, fields ...map[string]interface{}) {
	l.withFields(fields...).Warn(msg)
}
func (l *Logger) Error(msg string, fields ...map[string]interface{}) {
	l.withFields(fields...).Error(msg)
}
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

type fileHook struct {
	file      *os.File
	formatter logrus.Formatter
}

func (h *fileHook) Levels() []logrus.Level { return logrus.AllLevels }
func (h *fileHook) Fire(entry *logrus.Entry) error {
	line, err := h.formatter.Format(entry)
	if err != nil {
		return err
	}
	_, err = h.file.Write(line)
	return err
}
