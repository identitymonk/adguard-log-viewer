package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeTestConfig(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.txt")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestLoadConfig_ValidFile(t *testing.T) {
	path := writeTestConfig(t, "# comment\nlog_file = /var/log/query.json\nhttp_port = 9090\n")
	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.LogFilePath != "/var/log/query.json" {
		t.Errorf("LogFilePath = %q, want %q", cfg.LogFilePath, "/var/log/query.json")
	}
	if cfg.HTTPPort != 9090 {
		t.Errorf("HTTPPort = %d, want %d", cfg.HTTPPort, 9090)
	}
}

func TestLoadConfig_MissingFile(t *testing.T) {
	_, err := LoadConfig("/nonexistent/path/config.txt")
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
	if !strings.Contains(err.Error(), "cannot open config file") {
		t.Errorf("error should mention cannot open, got: %v", err)
	}
}

func TestLoadConfig_MalformedPort(t *testing.T) {
	path := writeTestConfig(t, "log_file = /tmp/log.json\nhttp_port = abc\n")
	_, err := LoadConfig(path)
	if err == nil {
		t.Fatal("expected error for malformed port, got nil")
	}
	if !strings.Contains(err.Error(), "valid integer") {
		t.Errorf("error should mention valid integer, got: %v", err)
	}
}

func TestLoadConfig_PortOutOfRange(t *testing.T) {
	path := writeTestConfig(t, "log_file = /tmp/log.json\nhttp_port = 70000\n")
	_, err := LoadConfig(path)
	if err == nil {
		t.Fatal("expected error for out-of-range port, got nil")
	}
	if !strings.Contains(err.Error(), "between 1 and 65535") {
		t.Errorf("error should mention port range, got: %v", err)
	}
}

func TestLoadConfig_MissingLogFile(t *testing.T) {
	path := writeTestConfig(t, "http_port = 8080\n")
	_, err := LoadConfig(path)
	if err == nil {
		t.Fatal("expected error for missing log_file, got nil")
	}
	if !strings.Contains(err.Error(), "missing required key 'log_file'") {
		t.Errorf("error should mention missing log_file, got: %v", err)
	}
}

func TestLoadConfig_MissingHTTPPort(t *testing.T) {
	path := writeTestConfig(t, "log_file = /tmp/log.json\n")
	_, err := LoadConfig(path)
	if err == nil {
		t.Fatal("expected error for missing http_port, got nil")
	}
	if !strings.Contains(err.Error(), "missing required key 'http_port'") {
		t.Errorf("error should mention missing http_port, got: %v", err)
	}
}

func TestLoadConfig_EmptyLogFilePath(t *testing.T) {
	path := writeTestConfig(t, "log_file = \nhttp_port = 8080\n")
	_, err := LoadConfig(path)
	if err == nil {
		t.Fatal("expected error for empty log_file, got nil")
	}
	if !strings.Contains(err.Error(), "must not be empty") {
		t.Errorf("error should mention empty log_file, got: %v", err)
	}
}

func TestLoadConfig_CommentsAndBlankLines(t *testing.T) {
	content := `# This is a comment

# Another comment
log_file = /data/querylog.json

http_port = 443
`
	path := writeTestConfig(t, content)
	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.LogFilePath != "/data/querylog.json" {
		t.Errorf("LogFilePath = %q, want %q", cfg.LogFilePath, "/data/querylog.json")
	}
	if cfg.HTTPPort != 443 {
		t.Errorf("HTTPPort = %d, want %d", cfg.HTTPPort, 443)
	}
}
