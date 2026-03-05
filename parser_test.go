package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func writeTempFile(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "querylog.json")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestParseLogFile_ExampleFile(t *testing.T) {
	entries, err := ParseLogFile("example/querylog.json", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("expected entries from example file, got 0")
	}

	// Verify first entry fields match the known first line.
	first := entries[0]
	wantTime, _ := time.Parse(time.RFC3339Nano, "2026-03-04T17:26:31.579969647-05:00")
	if !first.Timestamp.Equal(wantTime) {
		t.Errorf("first entry Timestamp = %v, want %v", first.Timestamp, wantTime)
	}
	if first.Hostname != "." {
		t.Errorf("first entry Hostname = %q, want %q", first.Hostname, ".")
	}
	if first.QueryType != "NS" {
		t.Errorf("first entry QueryType = %q, want %q", first.QueryType, "NS")
	}
	if first.ClientIP != "10.0.0.121" {
		t.Errorf("first entry ClientIP = %q, want %q", first.ClientIP, "10.0.0.121")
	}
	if first.IsFiltered {
		t.Error("first entry IsFiltered = true, want false")
	}
	if first.Elapsed != time.Duration(9483122) {
		t.Errorf("first entry Elapsed = %v, want %v", first.Elapsed, time.Duration(9483122))
	}

	// Verify third entry (index 2) is a filtered/blocked entry.
	if len(entries) < 3 {
		t.Fatalf("expected at least 3 entries, got %d", len(entries))
	}
	blocked := entries[2]
	if blocked.Hostname != "word-telemetry.officeapps.live.com" {
		t.Errorf("third entry Hostname = %q, want %q", blocked.Hostname, "word-telemetry.officeapps.live.com")
	}
	if !blocked.IsFiltered {
		t.Error("third entry IsFiltered = false, want true (blocked)")
	}
	if blocked.FilterRule != "||word-telemetry.officeapps.live.com^" {
		t.Errorf("third entry FilterRule = %q, want %q", blocked.FilterRule, "||word-telemetry.officeapps.live.com^")
	}
}

func TestParseLogFile_EmptyFile(t *testing.T) {
	path := writeTempFile(t, "")
	entries, err := ParseLogFile(path, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("expected 0 entries for empty file, got %d", len(entries))
	}
}

func TestParseLogFile_MalformedLines(t *testing.T) {
	content := `{"T":"2026-01-01T00:00:00Z","QH":"good.com","QT":"A","IP":"10.0.0.1","Result":{},"Elapsed":1000}
this is garbage
{"T":"2026-01-02T00:00:00Z","QH":"also-good.com","QT":"AAAA","IP":"10.0.0.2","Result":{},"Elapsed":2000}
{not valid json at all
{"T":"2026-01-03T00:00:00Z","QH":"third.com","QT":"A","IP":"10.0.0.3","Result":{},"Elapsed":3000}
`
	path := writeTempFile(t, content)
	entries, err := ParseLogFile(path, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 3 {
		t.Fatalf("expected 3 valid entries, got %d", len(entries))
	}
	if entries[0].Hostname != "good.com" {
		t.Errorf("entries[0].Hostname = %q, want %q", entries[0].Hostname, "good.com")
	}
	if entries[1].Hostname != "also-good.com" {
		t.Errorf("entries[1].Hostname = %q, want %q", entries[1].Hostname, "also-good.com")
	}
	if entries[2].Hostname != "third.com" {
		t.Errorf("entries[2].Hostname = %q, want %q", entries[2].Hostname, "third.com")
	}
}

func TestParseLogFile_MissingFields(t *testing.T) {
	content := `{"T":"2026-01-01T00:00:00Z"}
`
	path := writeTempFile(t, content)
	entries, err := ParseLogFile(path, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	e := entries[0]
	if e.Hostname != "" {
		t.Errorf("Hostname = %q, want empty string", e.Hostname)
	}
	if e.QueryType != "" {
		t.Errorf("QueryType = %q, want empty string", e.QueryType)
	}
	if e.ClientIP != "" {
		t.Errorf("ClientIP = %q, want empty string", e.ClientIP)
	}
	if e.IsFiltered {
		t.Error("IsFiltered = true, want false")
	}
	if e.Elapsed != 0 {
		t.Errorf("Elapsed = %v, want 0", e.Elapsed)
	}
	if e.Cached {
		t.Error("Cached = true, want false")
	}
	wantTime, _ := time.Parse(time.RFC3339Nano, "2026-01-01T00:00:00Z")
	if !e.Timestamp.Equal(wantTime) {
		t.Errorf("Timestamp = %v, want %v", e.Timestamp, wantTime)
	}
}
