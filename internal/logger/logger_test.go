package logger

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{"with valid path", "test.log", false},
		{"with empty path (disabled)", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.path != "" {
				tmpDir := t.TempDir()
				tt.path = filepath.Join(tmpDir, tt.path)
			}

			logger, err := New(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if logger == nil {
				t.Error("New() returned nil logger")
			}
			if tt.path != "" {
				logger.Close()
			}
		})
	}
}

func TestNew_NestedPath(t *testing.T) {
	tmpDir := t.TempDir()
	logDir := filepath.Join(tmpDir, "logs")
	logPath := filepath.Join(logDir, "test.log")

	// Create the parent directory
	if err := os.MkdirAll(logDir, 0755); err != nil {
		t.Fatalf("Failed to create log directory: %v", err)
	}

	logger, err := New(logPath)
	if err != nil {
		t.Errorf("New() with nested path error = %v", err)
		return
	}
	if logger == nil {
		t.Error("New() returned nil logger")
	}
	logger.Close()
}

func TestNew_InvalidPath(t *testing.T) {
	// Try to create a logger in a non-existent directory without proper permissions
	logger, err := New("/nonexistent/directory/test.log")
	if err == nil {
		t.Error("Expected error for invalid path, got nil")
		logger.Close()
	}
}

func TestLogger_Logging(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "test.log")

	logger, err := New(logPath)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Close()

	// Test various log levels
	logger.Info("Test info message: %d", 42)
	logger.Error("Test error message: %s", "error details")
	logger.Debug("Test debug message")
	logger.Log("Custom level message")

	// Read the log file
	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	logStr := string(content)
	if !strings.Contains(logStr, "INFO") {
		t.Error("Log file should contain INFO entry")
	}
	if !strings.Contains(logStr, "Test info message: 42") {
		t.Error("Log file should contain formatted info message")
	}
	if !strings.Contains(logStr, "ERROR") {
		t.Error("Log file should contain ERROR entry")
	}
	if !strings.Contains(logStr, "DEBUG") {
		t.Error("Log file should contain DEBUG entry")
	}
	if !strings.Contains(logStr, "Custom level message") {
		t.Error("Log file should contain custom log message")
	}
}

func TestLogger_Disabled(t *testing.T) {
	logger, err := New("")
	if err != nil {
		t.Fatalf("Failed to create disabled logger: %v", err)
	}

	// These should not panic even though logger is disabled
	logger.Info("This should not panic")
	logger.Error("This should not panic")
	logger.Debug("This should not panic")
	logger.Log("This should not panic")

	if logger.Enabled() {
		t.Error("Disabled logger should return false for Enabled()")
	}
}

func TestLogger_Enabled(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "test.log")

	logger, err := New(logPath)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Close()

	if !logger.Enabled() {
		t.Error("Active logger should return true for Enabled()")
	}
}

func TestLogger_Concurrent(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "test.log")

	logger, err := New(logPath)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Close()

	// Concurrent logging from multiple goroutines
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(id int) {
			for j := 0; j < 10; j++ {
				logger.Info("Goroutine %d, message %d", id, j)
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Read and verify log file
	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	lines := strings.Split(string(content), "\n")
	// Should have 100 lines (10 goroutines * 10 messages) + potentially empty line at end
	nonEmptyLines := 0
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			nonEmptyLines++
		}
	}
	if nonEmptyLines != 100 {
		t.Errorf("Expected 100 log lines, got %d", nonEmptyLines)
	}
}

func TestLogger_TimestampFormat(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "test.log")

	logger, err := New(logPath)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Close()

	logger.Info("Test message")

	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	// Check timestamp format: YYYY-MM-DD HH:MM:SS.mmm
	logStr := string(content)
	if !strings.Contains(logStr, "[20") {
		t.Error("Log should contain timestamp starting with [20")
	}
	if !strings.Contains(logStr, "]") {
		t.Error("Log should contain closing bracket for timestamp")
	}
}

func TestClose(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "test.log")

	logger, err := New(logPath)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	// Close should not error
	if err := logger.Close(); err != nil {
		t.Errorf("Close() returned error: %v", err)
	}
}

func TestClose_DisabledLogger(t *testing.T) {
	logger, err := New("")
	if err != nil {
		t.Fatalf("Failed to create disabled logger: %v", err)
	}

	// Close should not error for disabled logger
	if err := logger.Close(); err != nil {
		t.Errorf("Close() on disabled logger returned error: %v", err)
	}
}
