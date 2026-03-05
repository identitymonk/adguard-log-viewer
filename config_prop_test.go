package main

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"pgregory.net/rapid"
)

// Feature: adguard-log-viewer, Property 1: Config parse round-trip
// **Validates: Requirements 1.1, 1.2**
//
// For any valid log file path string and valid port number (1–65535),
// writing them to the config format and parsing the result with LoadConfig
// should yield a Config with the same log file path and port number.
func TestProperty_ConfigParseRoundTrip(t *testing.T) {
	dir := t.TempDir()

	rapid.Check(t, func(t *rapid.T) {
		// Generate a non-empty path that doesn't contain newlines or '=' to avoid ambiguity
		logFilePath := rapid.StringMatching(`[a-zA-Z0-9/_.\-]{1,200}`).Draw(t, "logFilePath")
		port := rapid.IntRange(1, 65535).Draw(t, "port")

		// Serialize to config format
		content := fmt.Sprintf("log_file = %s\nhttp_port = %d\n", logFilePath, port)

		// Write to temp file
		path := filepath.Join(dir, "config.txt")
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}

		// Parse
		cfg, err := LoadConfig(path)
		if err != nil {
			t.Fatalf("LoadConfig failed: %v", err)
		}

		// Verify equivalence
		if cfg.LogFilePath != logFilePath {
			t.Fatalf("LogFilePath = %q, want %q", cfg.LogFilePath, logFilePath)
		}
		if cfg.HTTPPort != port {
			t.Fatalf("HTTPPort = %d, want %d", cfg.HTTPPort, port)
		}
	})
}
