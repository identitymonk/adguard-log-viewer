package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"pgregory.net/rapid"
)

// Feature: adguard-log-viewer, Property 2: NDJSON parse round-trip
// **Validates: Requirements 2.1, 2.2**
//
// For any list of valid LogEntry values, serializing each to the NDJSON JSON
// format (one JSON object per line) and parsing the resulting file with
// ParseLogFile (with an accept-all filter) should produce a list of LogEntry
// values equivalent to the originals.
func TestNDJSONRoundTrip(t *testing.T) {
	dir := t.TempDir()

	genLogEntry := rapid.Custom(func(t *rapid.T) LogEntry {
		sec := rapid.Int64Range(0, 4102444800).Draw(t, "sec")   // 0 to ~2100
		nsec := rapid.Int64Range(0, 999999999).Draw(t, "nsec")
		ts := time.Unix(sec, nsec).UTC()

		hostname := rapid.StringMatching(`[a-z0-9\-]{1,50}(\.[a-z]{2,6}){0,3}`).Draw(t, "hostname")
		queryType := rapid.SampledFrom([]string{"A", "AAAA", "CNAME", "MX", "NS", "SRV", "TXT", "HTTPS", "SOA"}).Draw(t, "queryType")
		clientIP := fmt.Sprintf("%d.%d.%d.%d",
			rapid.IntRange(1, 255).Draw(t, "ip1"),
			rapid.IntRange(0, 255).Draw(t, "ip2"),
			rapid.IntRange(0, 255).Draw(t, "ip3"),
			rapid.IntRange(1, 254).Draw(t, "ip4"),
		)
		isFiltered := rapid.Bool().Draw(t, "isFiltered")
		filterRule := ""
		reason := 0
		if isFiltered {
			filterRule = rapid.StringMatching(`\|\|[a-z0-9\-]{1,30}\.[a-z]{2,6}\^`).Draw(t, "filterRule")
			reason = rapid.IntRange(1, 10).Draw(t, "reason")
		}
		elapsedNs := rapid.Int64Range(0, 5000000000).Draw(t, "elapsedNs") // 0 to 5s
		cached := rapid.Bool().Draw(t, "cached")

		return LogEntry{
			Timestamp:  ts,
			Hostname:   hostname,
			QueryType:  queryType,
			ClientIP:   clientIP,
			IsFiltered: isFiltered,
			FilterRule: filterRule,
			Reason:     reason,
			Elapsed:    time.Duration(elapsedNs),
			Cached:     cached,
		}
	})

	rapid.Check(t, func(t *rapid.T) {
		entries := rapid.SliceOfN(genLogEntry, 0, 50).Draw(t, "entries")

		// Serialize each LogEntry to NDJSON format
		var ndjson []byte
		for _, e := range entries {
			line := serializeLogEntry(e)
			ndjson = append(ndjson, line...)
			ndjson = append(ndjson, '\n')
		}

		// Write to temp file
		path := filepath.Join(dir, "roundtrip.json")
		if err := os.WriteFile(path, ndjson, 0644); err != nil {
			t.Fatal(err)
		}

		// Parse with accept-all filter
		parsed, err := ParseLogFile(path, nil)
		if err != nil {
			t.Fatalf("ParseLogFile failed: %v", err)
		}

		// Verify count
		if len(parsed) != len(entries) {
			t.Fatalf("got %d entries, want %d", len(parsed), len(entries))
		}

		// Verify equivalence
		for i := range entries {
			want := entries[i]
			got := parsed[i]

			if !got.Timestamp.Equal(want.Timestamp) {
				t.Fatalf("entry[%d] Timestamp = %v, want %v", i, got.Timestamp, want.Timestamp)
			}
			if got.Hostname != want.Hostname {
				t.Fatalf("entry[%d] Hostname = %q, want %q", i, got.Hostname, want.Hostname)
			}
			if got.QueryType != want.QueryType {
				t.Fatalf("entry[%d] QueryType = %q, want %q", i, got.QueryType, want.QueryType)
			}
			if got.ClientIP != want.ClientIP {
				t.Fatalf("entry[%d] ClientIP = %q, want %q", i, got.ClientIP, want.ClientIP)
			}
			if got.IsFiltered != want.IsFiltered {
				t.Fatalf("entry[%d] IsFiltered = %v, want %v", i, got.IsFiltered, want.IsFiltered)
			}
			if got.FilterRule != want.FilterRule {
				t.Fatalf("entry[%d] FilterRule = %q, want %q", i, got.FilterRule, want.FilterRule)
			}
			if got.Reason != want.Reason {
				t.Fatalf("entry[%d] Reason = %d, want %d", i, got.Reason, want.Reason)
			}
			if got.Elapsed != want.Elapsed {
				t.Fatalf("entry[%d] Elapsed = %v, want %v", i, got.Elapsed, want.Elapsed)
			}
			if got.Cached != want.Cached {
				t.Fatalf("entry[%d] Cached = %v, want %v", i, got.Cached, want.Cached)
			}
		}
	})
}

// serializeLogEntry converts a LogEntry to the NDJSON JSON format matching
// the rawLogEntry schema used by AdGuardHome.
func serializeLogEntry(e LogEntry) []byte {
	type ruleEntry struct {
		Text string `json:"Text"`
	}
	type resultObj struct {
		IsFiltered bool        `json:"IsFiltered,omitempty"`
		Reason     int         `json:"Reason,omitempty"`
		Rules      []ruleEntry `json:"Rules,omitempty"`
	}

	result := resultObj{}
	if e.IsFiltered || e.FilterRule != "" {
		result.IsFiltered = e.IsFiltered
		result.Reason = e.Reason
		if e.FilterRule != "" {
			result.Rules = []ruleEntry{{Text: e.FilterRule}}
		}
	}

	raw := struct {
		T       string          `json:"T"`
		QH      string          `json:"QH"`
		QT      string          `json:"QT"`
		IP      string          `json:"IP"`
		Result  json.RawMessage `json:"Result"`
		Elapsed int64           `json:"Elapsed"`
		Cached  bool            `json:"Cached"`
	}{
		T:       e.Timestamp.Format(time.RFC3339Nano),
		QH:      e.Hostname,
		QT:      e.QueryType,
		IP:      e.ClientIP,
		Elapsed: int64(e.Elapsed),
		Cached:  e.Cached,
	}

	resultBytes, _ := json.Marshal(result)
	raw.Result = resultBytes

	data, _ := json.Marshal(raw)
	return data
}

// Feature: adguard-log-viewer, Property 3: Malformed lines are skipped
// **Validates: Requirements 2.3**
//
// For any list of valid NDJSON log lines interspersed with arbitrary non-JSON
// strings, parsing the combined file should return exactly the entries
// corresponding to the valid lines, in order.
func TestMalformedLineSkipping(t *testing.T) {
	dir := t.TempDir()

	genLogEntry := rapid.Custom(func(t *rapid.T) LogEntry {
		sec := rapid.Int64Range(0, 4102444800).Draw(t, "sec")
		nsec := rapid.Int64Range(0, 999999999).Draw(t, "nsec")
		ts := time.Unix(sec, nsec).UTC()

		hostname := rapid.StringMatching(`[a-z0-9\-]{1,50}(\.[a-z]{2,6}){0,3}`).Draw(t, "hostname")
		queryType := rapid.SampledFrom([]string{"A", "AAAA", "CNAME", "MX", "NS", "SRV", "TXT", "HTTPS", "SOA"}).Draw(t, "queryType")
		clientIP := fmt.Sprintf("%d.%d.%d.%d",
			rapid.IntRange(1, 255).Draw(t, "ip1"),
			rapid.IntRange(0, 255).Draw(t, "ip2"),
			rapid.IntRange(0, 255).Draw(t, "ip3"),
			rapid.IntRange(1, 254).Draw(t, "ip4"),
		)
		isFiltered := rapid.Bool().Draw(t, "isFiltered")
		filterRule := ""
		reason := 0
		if isFiltered {
			filterRule = rapid.StringMatching(`\|\|[a-z0-9\-]{1,30}\.[a-z]{2,6}\^`).Draw(t, "filterRule")
			reason = rapid.IntRange(1, 10).Draw(t, "reason")
		}
		elapsedNs := rapid.Int64Range(0, 5000000000).Draw(t, "elapsedNs")
		cached := rapid.Bool().Draw(t, "cached")

		return LogEntry{
			Timestamp:  ts,
			Hostname:   hostname,
			QueryType:  queryType,
			ClientIP:   clientIP,
			IsFiltered: isFiltered,
			FilterRule: filterRule,
			Reason:     reason,
			Elapsed:    time.Duration(elapsedNs),
			Cached:     cached,
		}
	})

	// Generate garbage strings that are guaranteed not to be valid JSON objects.
	// We avoid lines starting with '{' to ensure they won't accidentally parse
	// as valid rawLogEntry JSON.
	genGarbage := rapid.Custom(func(t *rapid.T) string {
		prefix := rapid.SampledFrom([]string{
			"", "not json", "###", "ERROR:", "12345", "null", "[array]",
			"true", "false", "<html>", "-- comment --",
		}).Draw(t, "prefix")
		suffix := rapid.StringMatching(`[a-zA-Z0-9 !@#$%&*=+]{0,40}`).Draw(t, "suffix")
		return prefix + suffix
	})

	rapid.Check(t, func(t *rapid.T) {
		entries := rapid.SliceOfN(genLogEntry, 0, 30).Draw(t, "entries")
		garbageCount := rapid.IntRange(0, 20).Draw(t, "garbageCount")

		// Build the file content: intersperse garbage before, between, and after valid lines.
		var content []byte

		// Garbage lines before valid entries
		beforeCount := rapid.IntRange(0, garbageCount).Draw(t, "beforeCount")
		for i := 0; i < beforeCount; i++ {
			g := genGarbage.Draw(t, fmt.Sprintf("garbageBefore%d", i))
			content = append(content, []byte(g)...)
			content = append(content, '\n')
		}

		// Valid entries interspersed with garbage
		for i, e := range entries {
			line := serializeLogEntry(e)
			content = append(content, line...)
			content = append(content, '\n')

			// Optionally add garbage after each valid line
			if rapid.Bool().Draw(t, fmt.Sprintf("addGarbageAfter%d", i)) {
				g := genGarbage.Draw(t, fmt.Sprintf("garbageAfter%d", i))
				content = append(content, []byte(g)...)
				content = append(content, '\n')
			}
		}

		// Garbage lines after all valid entries
		afterCount := rapid.IntRange(0, 3).Draw(t, "afterCount")
		for i := 0; i < afterCount; i++ {
			g := genGarbage.Draw(t, fmt.Sprintf("garbageEnd%d", i))
			content = append(content, []byte(g)...)
			content = append(content, '\n')
		}

		// Write to temp file
		path := filepath.Join(dir, "malformed.json")
		if err := os.WriteFile(path, content, 0644); err != nil {
			t.Fatal(err)
		}

		// Parse with accept-all filter
		parsed, err := ParseLogFile(path, nil)
		if err != nil {
			t.Fatalf("ParseLogFile failed: %v", err)
		}

		// Verify count: only valid entries should be returned
		if len(parsed) != len(entries) {
			t.Fatalf("got %d entries, want %d", len(parsed), len(entries))
		}

		// Verify equivalence in order
		for i := range entries {
			want := entries[i]
			got := parsed[i]

			if !got.Timestamp.Equal(want.Timestamp) {
				t.Fatalf("entry[%d] Timestamp = %v, want %v", i, got.Timestamp, want.Timestamp)
			}
			if got.Hostname != want.Hostname {
				t.Fatalf("entry[%d] Hostname = %q, want %q", i, got.Hostname, want.Hostname)
			}
			if got.QueryType != want.QueryType {
				t.Fatalf("entry[%d] QueryType = %q, want %q", i, got.QueryType, want.QueryType)
			}
			if got.ClientIP != want.ClientIP {
				t.Fatalf("entry[%d] ClientIP = %q, want %q", i, got.ClientIP, want.ClientIP)
			}
			if got.IsFiltered != want.IsFiltered {
				t.Fatalf("entry[%d] IsFiltered = %v, want %v", i, got.IsFiltered, want.IsFiltered)
			}
			if got.FilterRule != want.FilterRule {
				t.Fatalf("entry[%d] FilterRule = %q, want %q", i, got.FilterRule, want.FilterRule)
			}
			if got.Reason != want.Reason {
				t.Fatalf("entry[%d] Reason = %d, want %d", i, got.Reason, want.Reason)
			}
			if got.Elapsed != want.Elapsed {
				t.Fatalf("entry[%d] Elapsed = %v, want %v", i, got.Elapsed, want.Elapsed)
			}
			if got.Cached != want.Cached {
				t.Fatalf("entry[%d] Cached = %v, want %v", i, got.Cached, want.Cached)
			}
		}
	})
}
