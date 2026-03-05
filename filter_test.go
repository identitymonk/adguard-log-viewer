package main

import (
	"net/url"
	"testing"
	"time"
)

func TestParseFilterParams_AllFields(t *testing.T) {
	query := url.Values{
		"ip":       {"192.168"},
		"hostname": {"example.com"},
		"start":    {"2024-01-15T10:30"},
		"end":      {"2024-01-15T18:00"},
		"status":   {"blocked"},
	}

	params := ParseFilterParams(query)

	if params.IP != "192.168" {
		t.Errorf("IP = %q, want %q", params.IP, "192.168")
	}
	if params.Hostname != "example.com" {
		t.Errorf("Hostname = %q, want %q", params.Hostname, "example.com")
	}
	if params.TimeStart == nil {
		t.Fatal("TimeStart is nil, want non-nil")
	}
	wantStart := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	if !params.TimeStart.Equal(wantStart) {
		t.Errorf("TimeStart = %v, want %v", params.TimeStart, wantStart)
	}
	if params.TimeEnd == nil {
		t.Fatal("TimeEnd is nil, want non-nil")
	}
	wantEnd := time.Date(2024, 1, 15, 18, 0, 0, 0, time.UTC)
	if !params.TimeEnd.Equal(wantEnd) {
		t.Errorf("TimeEnd = %v, want %v", params.TimeEnd, wantEnd)
	}
	if params.BlockStatus != "blocked" {
		t.Errorf("BlockStatus = %q, want %q", params.BlockStatus, "blocked")
	}
}

func TestParseFilterParams_EmptyQuery(t *testing.T) {
	params := ParseFilterParams(url.Values{})

	if params.IP != "" {
		t.Errorf("IP = %q, want empty", params.IP)
	}
	if params.Hostname != "" {
		t.Errorf("Hostname = %q, want empty", params.Hostname)
	}
	if params.TimeStart != nil {
		t.Errorf("TimeStart = %v, want nil", params.TimeStart)
	}
	if params.TimeEnd != nil {
		t.Errorf("TimeEnd = %v, want nil", params.TimeEnd)
	}
	if params.BlockStatus != "" {
		t.Errorf("BlockStatus = %q, want empty", params.BlockStatus)
	}
}

func TestParseFilterParams_InvalidTimeIgnored(t *testing.T) {
	query := url.Values{
		"start": {"not-a-date"},
		"end":   {"also-bad"},
	}

	params := ParseFilterParams(query)

	if params.TimeStart != nil {
		t.Errorf("TimeStart = %v, want nil for invalid time", params.TimeStart)
	}
	if params.TimeEnd != nil {
		t.Errorf("TimeEnd = %v, want nil for invalid time", params.TimeEnd)
	}
}

func TestParseFilterParams_InvalidStatusTreatedAsAll(t *testing.T) {
	query := url.Values{
		"status": {"invalid-status"},
	}

	params := ParseFilterParams(query)

	if params.BlockStatus != "" {
		t.Errorf("BlockStatus = %q, want empty for invalid status", params.BlockStatus)
	}
}

func TestParseFilterParams_AllowedStatus(t *testing.T) {
	query := url.Values{
		"status": {"allowed"},
	}

	params := ParseFilterParams(query)

	if params.BlockStatus != "allowed" {
		t.Errorf("BlockStatus = %q, want %q", params.BlockStatus, "allowed")
	}
}

// --- BuildFilter unit tests ---

func TestBuildFilter_IPSubstring(t *testing.T) {
	entries := []LogEntry{
		{ClientIP: "192.168.1.1"},
		{ClientIP: "192.168.1.20"},
		{ClientIP: "10.0.0.1"},
		{ClientIP: "172.16.0.1"},
	}

	tests := []struct {
		name    string
		ip      string
		wantIPs []string
	}{
		{"match prefix", "192.168", []string{"192.168.1.1", "192.168.1.20"}},
		{"match single", "10.0.0", []string{"10.0.0.1"}},
		{"match exact", "172.16.0.1", []string{"172.16.0.1"}},
		{"no match", "8.8.8", nil},
		{"partial digit", "1.20", []string{"192.168.1.20"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter := BuildFilter(FilterParams{IP: tt.ip})
			var got []string
			for _, e := range entries {
				if filter(e) {
					got = append(got, e.ClientIP)
				}
			}
			if len(got) != len(tt.wantIPs) {
				t.Fatalf("got %d entries %v, want %d entries %v", len(got), got, len(tt.wantIPs), tt.wantIPs)
			}
			for i, ip := range got {
				if ip != tt.wantIPs[i] {
					t.Errorf("entry[%d] IP = %q, want %q", i, ip, tt.wantIPs[i])
				}
			}
		})
	}
}

func TestBuildFilter_HostnameCaseInsensitive(t *testing.T) {
	entries := []LogEntry{
		{Hostname: "example.com"},
		{Hostname: "Example.COM"},
		{Hostname: "test.example.org"},
		{Hostname: "google.com"},
	}

	tests := []struct {
		name      string
		hostname  string
		wantHosts []string
	}{
		{"lowercase match", "example", []string{"example.com", "Example.COM", "test.example.org"}},
		{"uppercase query", "EXAMPLE", []string{"example.com", "Example.COM", "test.example.org"}},
		{"mixed case query", "ExAmPlE", []string{"example.com", "Example.COM", "test.example.org"}},
		{"exact domain", "google.com", []string{"google.com"}},
		{"no match", "yahoo", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter := BuildFilter(FilterParams{Hostname: tt.hostname})
			var got []string
			for _, e := range entries {
				if filter(e) {
					got = append(got, e.Hostname)
				}
			}
			if len(got) != len(tt.wantHosts) {
				t.Fatalf("got %d entries %v, want %d entries %v", len(got), got, len(tt.wantHosts), tt.wantHosts)
			}
			for i, h := range got {
				if h != tt.wantHosts[i] {
					t.Errorf("entry[%d] Hostname = %q, want %q", i, h, tt.wantHosts[i])
				}
			}
		})
	}
}

func TestBuildFilter_TimeRange(t *testing.T) {
	t1 := time.Date(2024, 1, 10, 8, 0, 0, 0, time.UTC)
	t2 := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)
	t3 := time.Date(2024, 1, 20, 18, 0, 0, 0, time.UTC)

	entries := []LogEntry{
		{Hostname: "early", Timestamp: t1},
		{Hostname: "mid", Timestamp: t2},
		{Hostname: "late", Timestamp: t3},
	}

	start := time.Date(2024, 1, 12, 0, 0, 0, 0, time.UTC)
	end := time.Date(2024, 1, 18, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name      string
		start     *time.Time
		end       *time.Time
		wantNames []string
	}{
		{"start only", &start, nil, []string{"mid", "late"}},
		{"end only", nil, &end, []string{"early", "mid"}},
		{"both", &start, &end, []string{"mid"}},
		{"neither", nil, nil, []string{"early", "mid", "late"}},
		{"exact boundary start", &t2, nil, []string{"mid", "late"}},
		{"exact boundary end", nil, &t2, []string{"early", "mid"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter := BuildFilter(FilterParams{TimeStart: tt.start, TimeEnd: tt.end})
			var got []string
			for _, e := range entries {
				if filter(e) {
					got = append(got, e.Hostname)
				}
			}
			if len(got) != len(tt.wantNames) {
				t.Fatalf("got %v, want %v", got, tt.wantNames)
			}
			for i, n := range got {
				if n != tt.wantNames[i] {
					t.Errorf("entry[%d] = %q, want %q", i, n, tt.wantNames[i])
				}
			}
		})
	}
}

func TestBuildFilter_BlockStatus(t *testing.T) {
	entries := []LogEntry{
		{Hostname: "blocked1", IsFiltered: true},
		{Hostname: "allowed1", IsFiltered: false},
		{Hostname: "blocked2", IsFiltered: true},
		{Hostname: "allowed2", IsFiltered: false},
	}

	tests := []struct {
		name      string
		status    string
		wantNames []string
	}{
		{"blocked", "blocked", []string{"blocked1", "blocked2"}},
		{"allowed", "allowed", []string{"allowed1", "allowed2"}},
		{"all (empty)", "", []string{"blocked1", "allowed1", "blocked2", "allowed2"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter := BuildFilter(FilterParams{BlockStatus: tt.status})
			var got []string
			for _, e := range entries {
				if filter(e) {
					got = append(got, e.Hostname)
				}
			}
			if len(got) != len(tt.wantNames) {
				t.Fatalf("got %v, want %v", got, tt.wantNames)
			}
			for i, n := range got {
				if n != tt.wantNames[i] {
					t.Errorf("entry[%d] = %q, want %q", i, n, tt.wantNames[i])
				}
			}
		})
	}
}

func TestBuildFilter_NoFilters(t *testing.T) {
	entries := []LogEntry{
		{Hostname: "a.com", ClientIP: "1.1.1.1", IsFiltered: true},
		{Hostname: "b.com", ClientIP: "2.2.2.2", IsFiltered: false},
		{Hostname: "c.com", ClientIP: "3.3.3.3", IsFiltered: true},
	}

	filter := BuildFilter(FilterParams{})
	for i, e := range entries {
		if !filter(e) {
			t.Errorf("entry[%d] (%s) was excluded with no filters active", i, e.Hostname)
		}
	}
}

func TestBuildFilter_Combined(t *testing.T) {
	t1 := time.Date(2024, 1, 10, 8, 0, 0, 0, time.UTC)
	t2 := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)
	t3 := time.Date(2024, 1, 20, 18, 0, 0, 0, time.UTC)

	start := time.Date(2024, 1, 12, 0, 0, 0, 0, time.UTC)
	end := time.Date(2024, 1, 18, 0, 0, 0, 0, time.UTC)

	entries := []LogEntry{
		{Hostname: "ads.example.com", ClientIP: "192.168.1.10", Timestamp: t1, IsFiltered: true},
		{Hostname: "ads.example.com", ClientIP: "192.168.1.10", Timestamp: t2, IsFiltered: true},
		{Hostname: "safe.example.com", ClientIP: "192.168.1.10", Timestamp: t2, IsFiltered: false},
		{Hostname: "ads.example.com", ClientIP: "10.0.0.1", Timestamp: t2, IsFiltered: true},
		{Hostname: "ads.example.com", ClientIP: "192.168.1.10", Timestamp: t3, IsFiltered: true},
	}

	// Filter: IP contains "192.168", hostname contains "ads", time in range, blocked only
	filter := BuildFilter(FilterParams{
		IP:          "192.168",
		Hostname:    "ads",
		TimeStart:   &start,
		TimeEnd:     &end,
		BlockStatus: "blocked",
	})

	var got []int
	for i, e := range entries {
		if filter(e) {
			got = append(got, i)
		}
	}

	// Only entry[1] matches all criteria:
	// - IP "192.168.1.10" contains "192.168" ✓
	// - Hostname "ads.example.com" contains "ads" ✓
	// - Timestamp t2 is within [start, end] ✓
	// - IsFiltered=true matches "blocked" ✓
	want := []int{1}
	if len(got) != len(want) {
		t.Fatalf("got indices %v, want %v", got, want)
	}
	for i, idx := range got {
		if idx != want[i] {
			t.Errorf("got[%d] = %d, want %d", i, idx, want[i])
		}
	}
}
