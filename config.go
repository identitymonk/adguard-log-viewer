package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Config holds the application configuration.
type Config struct {
	LogFilePath string
	HTTPPort    int
}

// LoadConfig reads the config file from the given path and returns a Config.
// The config file uses a simple "key = value" format, one per line.
// Lines starting with '#' are comments and are ignored.
// Returns an error if the file is missing, unreadable, or has invalid content.
func LoadConfig(path string) (Config, error) {
	f, err := os.Open(path)
	if err != nil {
		return Config{}, fmt.Errorf("cannot open config file %q: %w\nExpected format:\n  log_file = /path/to/querylog.json\n  http_port = 8080", path, err)
	}
	defer f.Close()

	cfg := Config{}
	foundLogFile := false
	foundPort := false

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		switch key {
		case "log_file":
			if value == "" {
				return Config{}, fmt.Errorf("config: log_file must not be empty")
			}
			cfg.LogFilePath = value
			foundLogFile = true
		case "http_port":
			port, err := strconv.Atoi(value)
			if err != nil {
				return Config{}, fmt.Errorf("config: http_port must be a valid integer, got %q", value)
			}
			if port < 1 || port > 65535 {
				return Config{}, fmt.Errorf("config: http_port must be between 1 and 65535, got %d", port)
			}
			cfg.HTTPPort = port
			foundPort = true
		}
	}

	if err := scanner.Err(); err != nil {
		return Config{}, fmt.Errorf("error reading config file %q: %w", path, err)
	}

	if !foundLogFile {
		return Config{}, fmt.Errorf("config: missing required key 'log_file'")
	}
	if !foundPort {
		return Config{}, fmt.Errorf("config: missing required key 'http_port'")
	}

	return cfg, nil
}
