package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

// setupHandler creates a handler using the example log file and the real template.
func setupHandler(t *testing.T) http.HandlerFunc {
	t.Helper()
	tmpl, err := LoadTemplate("template.html")
	if err != nil {
		t.Fatalf("LoadTemplate: %v", err)
	}
	cfg := Config{LogFilePath: "example/querylog.json", HTTPPort: 8080}
	return NewHandler(cfg, tmpl)
}

func TestHandler_BasicPage(t *testing.T) {
	handler := setupHandler(t)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "AdGuard Log Viewer") {
		t.Error("response missing title 'AdGuard Log Viewer'")
	}
	if !strings.Contains(body, "amazon.com") {
		t.Error("response missing 'amazon.com' entries")
	}
	if !strings.Contains(body, "10.0.0.121") {
		t.Error("response missing '10.0.0.121' entries")
	}
}

func TestHandler_FilterByIP(t *testing.T) {
	handler := setupHandler(t)
	req := httptest.NewRequest(http.MethodGet, "/?ip=10.0.0.149", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "10.0.0.149") {
		t.Error("response missing '10.0.0.149'")
	}
	if !strings.Contains(body, "fxpsbs-na.amazon.com") {
		t.Error("response missing 'fxpsbs-na.amazon.com'")
	}
	// Verify other IPs are filtered out from data rows.
	// The IP value "10.0.0.121" should not appear in table data cells.
	// It may appear in the filter form input, so check specifically in <tr> context.
	for _, line := range strings.Split(body, "\n") {
		if strings.Contains(line, "<tr") && strings.Contains(line, "10.0.0.121") {
			t.Error("response should not contain '10.0.0.121' in a table row when filtering by IP 10.0.0.149")
			break
		}
	}
}

func TestHandler_FilterByHostname(t *testing.T) {
	handler := setupHandler(t)
	req := httptest.NewRequest(http.MethodGet, "/?hostname=amazon", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "amazon") {
		t.Error("response missing 'amazon' entries")
	}
	// officeapps entries should be filtered out
	for _, line := range strings.Split(body, "\n") {
		if strings.Contains(line, "<tr") && strings.Contains(line, "officeapps") {
			t.Error("response should not contain 'officeapps' entries when filtering by hostname 'amazon'")
			break
		}
	}
}

func TestHandler_FilterByStatus(t *testing.T) {
	handler := setupHandler(t)
	req := httptest.NewRequest(http.MethodGet, "/?status=blocked", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "word-telemetry.officeapps.live.com") {
		t.Error("response missing 'word-telemetry.officeapps.live.com'")
	}
	if !strings.Contains(body, "Blocked") {
		t.Error("response missing 'Blocked' status label")
	}
}

func TestHandler_FilterByStatusAllowed(t *testing.T) {
	handler := setupHandler(t)
	req := httptest.NewRequest(http.MethodGet, "/?status=allowed", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	body := rec.Body.String()
	// word-telemetry is blocked, so it should not appear in data rows
	for _, line := range strings.Split(body, "\n") {
		if strings.Contains(line, "<tr") && strings.Contains(line, "word-telemetry.officeapps.live.com") {
			t.Error("response should not contain 'word-telemetry.officeapps.live.com' in a data row when filtering by status=allowed")
			break
		}
	}
}

func TestHandler_MissingLogFile(t *testing.T) {
	tmpl, err := LoadTemplate("template.html")
	if err != nil {
		t.Fatalf("LoadTemplate: %v", err)
	}
	cfg := Config{LogFilePath: "/nonexistent/path/querylog.json", HTTPPort: 8080}
	handler := NewHandler(cfg, tmpl)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "Error reading log file") {
		t.Error("response missing 'Error reading log file' message")
	}
}

func TestHandler_EmptyLogFile(t *testing.T) {
	tmpl, err := LoadTemplate("template.html")
	if err != nil {
		t.Fatalf("LoadTemplate: %v", err)
	}

	tmpFile, err := os.CreateTemp("", "empty-log-*.json")
	if err != nil {
		t.Fatalf("CreateTemp: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	cfg := Config{LogFilePath: tmpFile.Name(), HTTPPort: 8080}
	handler := NewHandler(cfg, tmpl)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "No log entries found.") {
		t.Error("response missing 'No log entries found.' message")
	}
}
