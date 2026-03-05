package main

import (
	"bufio"
	"encoding/json"
	"os"
	"time"
)

// toLogEntry converts a rawLogEntry into the domain LogEntry model.
// It parses the RFC3339Nano timestamp, converts nanosecond elapsed to
// time.Duration, and unmarshals the Result JSON into filter fields.
func toLogEntry(raw rawLogEntry) (LogEntry, error) {
	ts, err := time.Parse(time.RFC3339Nano, raw.T)
	if err != nil {
		return LogEntry{}, err
	}

	entry := LogEntry{
		Timestamp: ts,
		Hostname:  raw.QH,
		QueryType: raw.QT,
		ClientIP:  raw.IP,
		Elapsed:   time.Duration(raw.Elapsed),
		Cached:    raw.Cached,
	}

	// Unmarshal Result if present and non-empty.
	if len(raw.Result) > 0 && string(raw.Result) != "{}" {
		var res rawResult
		if err := json.Unmarshal(raw.Result, &res); err == nil {
			entry.IsFiltered = res.IsFiltered
			entry.Reason = res.Reason
			if len(res.Rules) > 0 {
				entry.FilterRule = res.Rules[0].Text
			}
		}
	}

	return entry, nil
}

// ParseLogFile opens the file at path and parses each line as a LogEntry.
// Malformed lines are skipped. Returns entries in file order (oldest first).
// Accepts a filter function to discard non-matching entries during parsing,
// avoiding accumulation of unneeded entries in memory.
// If filter is nil, all entries are accepted.
func ParseLogFile(path string, filter func(LogEntry) bool) ([]LogEntry, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var entries []LogEntry
	scanner := bufio.NewScanner(f)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var raw rawLogEntry
		if err := json.Unmarshal(line, &raw); err != nil {
			// Malformed JSON — skip silently.
			continue
		}

		entry, err := toLogEntry(raw)
		if err != nil {
			// Bad timestamp or conversion — skip silently.
			continue
		}

		if filter != nil && !filter(entry) {
			continue
		}

		entries = append(entries, entry)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	if entries == nil {
		entries = []LogEntry{}
	}

	return entries, nil
}
