package main

import (
	"encoding/json"
	"time"
)

// LogEntry is the domain model for a parsed DNS query log entry.
type LogEntry struct {
	Timestamp  time.Time
	Hostname   string
	QueryType  string
	ClientIP   string
	IsFiltered bool
	FilterRule string
	Reason     int
	Elapsed    time.Duration
	Cached     bool
}

// rawLogEntry maps directly to the NDJSON schema produced by AdGuardHome.
type rawLogEntry struct {
	T       string          `json:"T"`
	QH      string          `json:"QH"`
	QT      string          `json:"QT"`
	IP      string          `json:"IP"`
	Result  json.RawMessage `json:"Result"`
	Elapsed int64           `json:"Elapsed"`
	Cached  bool            `json:"Cached"`
}

// rawResult maps the Result object inside a raw log entry.
type rawResult struct {
	IsFiltered bool `json:"IsFiltered"`
	Reason     int  `json:"Reason"`
	Rules      []struct {
		Text string `json:"Text"`
	} `json:"Rules"`
}
