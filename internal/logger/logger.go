package logger

import (
	"fmt"
	"io"
	"os"
	"sync"
	"time"
)

// Logger provides thread-safe file-based logging for game actions.
type Logger struct {
	mu     sync.Mutex
	file   *os.File
	writer io.Writer
}

// New creates a new Logger that writes to the specified file path.
// If path is empty, logging is disabled (no-op logger).
func New(path string) (*Logger, error) {
	if path == "" {
		return &Logger{}, nil
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}
	return &Logger{file: file, writer: file}, nil
}

// Close closes the underlying log file.
func (l *Logger) Close() error {
	if l.file == nil {
		return nil
	}
	return l.file.Close()
}

// Log writes a timestamped message to the log file.
func (l *Logger) Log(format string, args ...interface{}) {
	if l.writer == nil {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()

	timestamp := time.Now().Format("2006-01-02 15:04:05.000")
	message := fmt.Sprintf(format, args...)
	fmt.Fprintf(l.writer, "[%s] %s\n", timestamp, message)
}

// Info logs an informational message.
func (l *Logger) Info(format string, args ...interface{}) {
	l.Log("INFO  "+format, args...)
}

// Error logs an error message.
func (l *Logger) Error(format string, args ...interface{}) {
	l.Log("ERROR "+format, args...)
}

// Debug logs a debug message.
func (l *Logger) Debug(format string, args ...interface{}) {
	l.Log("DEBUG "+format, args...)
}

// Enabled returns true if logging is active.
func (l *Logger) Enabled() bool {
	return l.writer != nil
}
