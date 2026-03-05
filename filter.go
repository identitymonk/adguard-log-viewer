package main

import (
	"net/url"
	"strings"
	"time"
)

// datetimeLocalFormat is the time format used by HTML datetime-local inputs.
const datetimeLocalFormat = "2006-01-02T15:04"

// FilterParams holds the parsed filter criteria from URL query parameters.
type FilterParams struct {
	IP          string
	Hostname    string
	TimeStart   *time.Time
	TimeEnd     *time.Time
	BlockStatus string // "blocked", "allowed", or "" (all)
}

// ParseFilterParams extracts FilterParams from URL query parameters.
// Invalid time values are silently ignored (treated as not set).
// Invalid block status values are treated as "" (all).
func ParseFilterParams(query url.Values) FilterParams {
	params := FilterParams{
		IP:       query.Get("ip"),
		Hostname: query.Get("hostname"),
	}

	if s := query.Get("start"); s != "" {
		if t, err := time.Parse(datetimeLocalFormat, s); err == nil {
			params.TimeStart = &t
		}
	}

	if s := query.Get("end"); s != "" {
		if t, err := time.Parse(datetimeLocalFormat, s); err == nil {
			params.TimeEnd = &t
		}
	}

	switch query.Get("status") {
	case "blocked", "allowed":
		params.BlockStatus = query.Get("status")
	default:
		params.BlockStatus = ""
	}

	return params
}

// BuildFilter returns a filter function that returns true for entries
// matching ALL active filter criteria (logical AND).
func BuildFilter(params FilterParams) func(LogEntry) bool {
	return func(entry LogEntry) bool {
		if params.IP != "" && !strings.Contains(entry.ClientIP, params.IP) {
			return false
		}
		if params.Hostname != "" && !strings.Contains(strings.ToLower(entry.Hostname), strings.ToLower(params.Hostname)) {
			return false
		}
		if params.TimeStart != nil && entry.Timestamp.Before(*params.TimeStart) {
			return false
		}
		if params.TimeEnd != nil && entry.Timestamp.After(*params.TimeEnd) {
			return false
		}
		switch params.BlockStatus {
		case "blocked":
			if !entry.IsFiltered {
				return false
			}
		case "allowed":
			if entry.IsFiltered {
				return false
			}
		}
		return true
	}
}
