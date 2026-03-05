package main

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"pgregory.net/rapid"
)

// Feature: adguard-log-viewer, Property 8: IP substring filter correctness
// **Validates: Requirements 4.1, 4.3**
//
// For any list of LogEntry values and any non-empty IP filter string,
// every entry returned by the IP filter should have a client IP containing
// the filter string as a substring, and every entry excluded should NOT
// contain the filter string.
func TestProperty_IPSubstringFilter(t *testing.T) {
	genOctet := rapid.IntRange(0, 255)
	genIP := rapid.Custom(func(t *rapid.T) string {
		return fmt.Sprintf("%d.%d.%d.%d",
			genOctet.Draw(t, "o1"),
			genOctet.Draw(t, "o2"),
			genOctet.Draw(t, "o3"),
			genOctet.Draw(t, "o4"),
		)
	})

	genEntry := rapid.Custom(func(t *rapid.T) LogEntry {
		return LogEntry{
			ClientIP: genIP.Draw(t, "clientIP"),
		}
	})

	rapid.Check(t, func(t *rapid.T) {
		entries := rapid.SliceOfN(genEntry, 1, 50).Draw(t, "entries")

		// Generate a non-empty IP substring: pick from a random IP-like fragment
		// Use a substring of a generated IP to ensure realistic filter values
		sourceIP := genIP.Draw(t, "sourceIP")
		start := rapid.IntRange(0, len(sourceIP)-1).Draw(t, "start")
		end := rapid.IntRange(start+1, len(sourceIP)).Draw(t, "end")
		ipSubstring := sourceIP[start:end]

		filter := BuildFilter(FilterParams{IP: ipSubstring})

		for _, entry := range entries {
			passed := filter(entry)
			contains := strings.Contains(entry.ClientIP, ipSubstring)

			if passed && !contains {
				t.Fatalf("entry with ClientIP=%q passed filter IP=%q but does not contain substring",
					entry.ClientIP, ipSubstring)
			}
			if !passed && contains {
				t.Fatalf("entry with ClientIP=%q excluded by filter IP=%q but contains substring",
					entry.ClientIP, ipSubstring)
			}
		}
	})
}

// Feature: adguard-log-viewer, Property 9: Hostname case-insensitive substring filter
// **Validates: Requirements 5.1**
//
// For any list of LogEntry values and any non-empty hostname filter string,
// every entry returned by the hostname filter should have a hostname containing
// the filter string as a case-insensitive substring, and every entry excluded
// should NOT contain it.
func TestProperty_HostnameSubstringFilter(t *testing.T) {
	tlds := []string{".com", ".org", ".net", ".io", ".dev"}

	genLabel := rapid.StringMatching(`[a-zA-Z][a-zA-Z0-9]{0,9}`)

	genHostname := rapid.Custom(func(t *rapid.T) string {
		numLabels := rapid.IntRange(1, 3).Draw(t, "numLabels")
		parts := make([]string, numLabels)
		for i := 0; i < numLabels; i++ {
			parts[i] = genLabel.Draw(t, fmt.Sprintf("label%d", i))
		}
		tld := tlds[rapid.IntRange(0, len(tlds)-1).Draw(t, "tldIdx")]
		return strings.Join(parts, ".") + tld
	})

	genEntry := rapid.Custom(func(t *rapid.T) LogEntry {
		return LogEntry{
			Hostname: genHostname.Draw(t, "hostname"),
		}
	})

	rapid.Check(t, func(t *rapid.T) {
		entries := rapid.SliceOfN(genEntry, 1, 50).Draw(t, "entries")

		// Generate a non-empty hostname substring from a random hostname
		sourceHostname := genHostname.Draw(t, "sourceHostname")
		start := rapid.IntRange(0, len(sourceHostname)-1).Draw(t, "start")
		end := rapid.IntRange(start+1, len(sourceHostname)).Draw(t, "end")
		hostnameSubstring := sourceHostname[start:end]

		filter := BuildFilter(FilterParams{Hostname: hostnameSubstring})

		for _, entry := range entries {
			passed := filter(entry)
			contains := strings.Contains(strings.ToLower(entry.Hostname), strings.ToLower(hostnameSubstring))

			if passed && !contains {
				t.Fatalf("entry with Hostname=%q passed filter Hostname=%q but does not contain substring (case-insensitive)",
					entry.Hostname, hostnameSubstring)
			}
			if !passed && contains {
				t.Fatalf("entry with Hostname=%q excluded by filter Hostname=%q but contains substring (case-insensitive)",
					entry.Hostname, hostnameSubstring)
			}
		}
	})
}

// Feature: adguard-log-viewer, Property 10: Time range filter correctness
// **Validates: Requirements 6.1, 6.2, 6.3**
//
// For any list of LogEntry values and any start/end time pair where start ≤ end,
// every entry returned by the time range filter should have a timestamp t where
// start ≤ t ≤ end, and every excluded entry should fall outside that range.
func TestProperty_TimeRangeFilter(t *testing.T) {
	// Time range: 2020-01-01 to 2026-01-01 in Unix seconds
	minUnix := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC).Unix()
	maxUnix := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC).Unix()

	// Generator for a time truncated to minute precision (datetime-local format)
	genTime := rapid.Custom(func(t *rapid.T) time.Time {
		unix := rapid.Int64Range(minUnix, maxUnix).Draw(t, "unix")
		return time.Unix(unix, 0).UTC().Truncate(time.Minute)
	})

	genEntry := rapid.Custom(func(t *rapid.T) LogEntry {
		return LogEntry{
			Timestamp: genTime.Draw(t, "timestamp"),
		}
	})

	rapid.Check(t, func(t *rapid.T) {
		entries := rapid.SliceOfN(genEntry, 1, 50).Draw(t, "entries")

		// Generate two times and use the earlier as start, later as end
		t1 := genTime.Draw(t, "t1")
		t2 := genTime.Draw(t, "t2")
		var start, end time.Time
		if t1.Before(t2) {
			start, end = t1, t2
		} else {
			start, end = t2, t1
		}

		filter := BuildFilter(FilterParams{TimeStart: &start, TimeEnd: &end})

		for _, entry := range entries {
			passed := filter(entry)
			inRange := !entry.Timestamp.Before(start) && !entry.Timestamp.After(end)

			if passed && !inRange {
				t.Fatalf("entry with Timestamp=%v passed filter [%v, %v] but is outside range",
					entry.Timestamp, start, end)
			}
			if !passed && inRange {
				t.Fatalf("entry with Timestamp=%v excluded by filter [%v, %v] but is within range",
					entry.Timestamp, start, end)
			}
		}
	})
}

// Feature: adguard-log-viewer, Property 11: Block status filter correctness
// **Validates: Requirements 7.1, 7.2, 7.4**
//
// For any list of LogEntry values and any block status filter value
// ("blocked", "allowed", or ""), the returned entries should satisfy:
// if "blocked", all have IsFiltered=true; if "allowed", all have
// IsFiltered=false; if empty, all entries are returned.
func TestProperty_BlockStatusFilter(t *testing.T) {
	genEntry := rapid.Custom(func(t *rapid.T) LogEntry {
		return LogEntry{
			IsFiltered: rapid.Bool().Draw(t, "isFiltered"),
			Hostname:   fmt.Sprintf("host%d.example.com", rapid.IntRange(0, 100).Draw(t, "hostNum")),
			ClientIP:   fmt.Sprintf("10.0.0.%d", rapid.IntRange(1, 254).Draw(t, "ipOctet")),
		}
	})

	statuses := []string{"blocked", "allowed", ""}

	rapid.Check(t, func(t *rapid.T) {
		entries := rapid.SliceOfN(genEntry, 1, 50).Draw(t, "entries")
		status := statuses[rapid.IntRange(0, len(statuses)-1).Draw(t, "statusIdx")]

		filter := BuildFilter(FilterParams{BlockStatus: status})

		for _, entry := range entries {
			passed := filter(entry)

			switch status {
			case "blocked":
				if passed && !entry.IsFiltered {
					t.Fatalf("entry with IsFiltered=%v passed 'blocked' filter but should only pass when IsFiltered=true",
						entry.IsFiltered)
				}
				if !passed && entry.IsFiltered {
					t.Fatalf("entry with IsFiltered=%v excluded by 'blocked' filter but should pass when IsFiltered=true",
						entry.IsFiltered)
				}
			case "allowed":
				if passed && entry.IsFiltered {
					t.Fatalf("entry with IsFiltered=%v passed 'allowed' filter but should only pass when IsFiltered=false",
						entry.IsFiltered)
				}
				if !passed && !entry.IsFiltered {
					t.Fatalf("entry with IsFiltered=%v excluded by 'allowed' filter but should pass when IsFiltered=false",
						entry.IsFiltered)
				}
			case "":
				if !passed {
					t.Fatalf("entry excluded by empty status filter, but all entries should pass when status is empty")
				}
			}
		}
	})
}

// Feature: adguard-log-viewer, Property 12: Composite AND filtering
// **Validates: Requirements 8.1**
//
// For any list of LogEntry values and any combination of filter parameters
// (IP, hostname, time range, block status), the set of entries returned by
// the composite filter should equal the intersection of the sets returned
// by each individual filter applied independently.
func TestProperty_CompositeANDFilter(t *testing.T) {
	// Time range: 2020-01-01 to 2026-01-01 in Unix seconds
	minUnix := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC).Unix()
	maxUnix := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC).Unix()

	genOctet := rapid.IntRange(0, 255)
	genIP := rapid.Custom(func(t *rapid.T) string {
		return fmt.Sprintf("%d.%d.%d.%d",
			genOctet.Draw(t, "o1"),
			genOctet.Draw(t, "o2"),
			genOctet.Draw(t, "o3"),
			genOctet.Draw(t, "o4"),
		)
	})

	tlds := []string{".com", ".org", ".net", ".io", ".dev"}
	genLabel := rapid.StringMatching(`[a-zA-Z][a-zA-Z0-9]{0,5}`)
	genHostname := rapid.Custom(func(t *rapid.T) string {
		label := genLabel.Draw(t, "label")
		tld := tlds[rapid.IntRange(0, len(tlds)-1).Draw(t, "tldIdx")]
		return label + tld
	})

	genTime := rapid.Custom(func(t *rapid.T) time.Time {
		unix := rapid.Int64Range(minUnix, maxUnix).Draw(t, "unix")
		return time.Unix(unix, 0).UTC().Truncate(time.Minute)
	})

	genEntry := rapid.Custom(func(t *rapid.T) LogEntry {
		return LogEntry{
			ClientIP:   genIP.Draw(t, "clientIP"),
			Hostname:   genHostname.Draw(t, "hostname"),
			Timestamp:  genTime.Draw(t, "timestamp"),
			IsFiltered: rapid.Bool().Draw(t, "isFiltered"),
		}
	})

	statuses := []string{"blocked", "allowed", ""}

	rapid.Check(t, func(t *rapid.T) {
		entries := rapid.SliceOfN(genEntry, 1, 50).Draw(t, "entries")

		// Generate random FilterParams with all fields potentially set
		// IP: sometimes empty, sometimes a substring from a random IP
		var ipFilter string
		if rapid.Bool().Draw(t, "hasIP") {
			src := genIP.Draw(t, "srcIP")
			s := rapid.IntRange(0, len(src)-1).Draw(t, "ipStart")
			e := rapid.IntRange(s+1, len(src)).Draw(t, "ipEnd")
			ipFilter = src[s:e]
		}

		// Hostname: sometimes empty, sometimes a substring from a random hostname
		var hostnameFilter string
		if rapid.Bool().Draw(t, "hasHostname") {
			src := genHostname.Draw(t, "srcHostname")
			s := rapid.IntRange(0, len(src)-1).Draw(t, "hnStart")
			e := rapid.IntRange(s+1, len(src)).Draw(t, "hnEnd")
			hostnameFilter = src[s:e]
		}

		// Time range: sometimes nil, sometimes set (ensure start <= end)
		var timeStart, timeEnd *time.Time
		if rapid.Bool().Draw(t, "hasTime") {
			t1 := genTime.Draw(t, "rangeT1")
			t2 := genTime.Draw(t, "rangeT2")
			if t1.After(t2) {
				t1, t2 = t2, t1
			}
			timeStart = &t1
			timeEnd = &t2
		}

		// Block status: one of "blocked", "allowed", ""
		blockStatus := statuses[rapid.IntRange(0, len(statuses)-1).Draw(t, "statusIdx")]

		// Composite filter params
		compositeParams := FilterParams{
			IP:          ipFilter,
			Hostname:    hostnameFilter,
			TimeStart:   timeStart,
			TimeEnd:     timeEnd,
			BlockStatus: blockStatus,
		}

		// Build composite filter (all params at once)
		compositeFilter := BuildFilter(compositeParams)

		// Build individual filters
		ipOnlyFilter := BuildFilter(FilterParams{IP: compositeParams.IP})
		hostnameOnlyFilter := BuildFilter(FilterParams{Hostname: compositeParams.Hostname})
		timeOnlyFilter := BuildFilter(FilterParams{TimeStart: compositeParams.TimeStart, TimeEnd: compositeParams.TimeEnd})
		statusOnlyFilter := BuildFilter(FilterParams{BlockStatus: compositeParams.BlockStatus})

		// Verify: composite passes iff ALL individual filters pass (intersection)
		for i, entry := range entries {
			compositeResult := compositeFilter(entry)
			ipResult := ipOnlyFilter(entry)
			hostnameResult := hostnameOnlyFilter(entry)
			timeResult := timeOnlyFilter(entry)
			statusResult := statusOnlyFilter(entry)

			intersectionResult := ipResult && hostnameResult && timeResult && statusResult

			if compositeResult != intersectionResult {
				t.Fatalf("entry[%d]: composite=%v but intersection of individual filters=%v "+
					"(ip=%v, hostname=%v, time=%v, status=%v) "+
					"params={IP:%q, Hostname:%q, TimeStart:%v, TimeEnd:%v, BlockStatus:%q} "+
					"entry={ClientIP:%q, Hostname:%q, Timestamp:%v, IsFiltered:%v}",
					i, compositeResult, intersectionResult,
					ipResult, hostnameResult, timeResult, statusResult,
					compositeParams.IP, compositeParams.Hostname, compositeParams.TimeStart, compositeParams.TimeEnd, compositeParams.BlockStatus,
					entry.ClientIP, entry.Hostname, entry.Timestamp, entry.IsFiltered)
			}
		}
	})
}
