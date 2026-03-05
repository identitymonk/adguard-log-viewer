package main

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
	"time"

	"pgregory.net/rapid"
)

// Feature: adguard-log-viewer, Property 4: Rendered HTML contains all entry fields
// **Validates: Requirements 3.1**
//
// For any LogEntry, the rendered HTML table row for that entry should contain
// the entry's timestamp, hostname, query type, client IP, filter status, and
// cached status as substrings.
func TestProperty_RenderedHTMLContainsFields(t *testing.T) {
	tmpl, err := LoadTemplate("template.html")
	if err != nil {
		t.Fatalf("LoadTemplate: %v", err)
	}

	// Time range: 2020-01-01 to 2026-01-01 in Unix seconds
	minUnix := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC).Unix()
	maxUnix := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC).Unix()

	queryTypes := []string{"A", "AAAA", "HTTPS", "SRV", "MX", "TXT", "CNAME"}

	rapid.Check(t, func(t *rapid.T) {
		// Generate between 1 and 10 random entries
		n := rapid.IntRange(1, 10).Draw(t, "numEntries")
		entries := make([]LogEntry, n)

		for i := 0; i < n; i++ {
			// Random timestamp truncated to seconds
			ts := time.Unix(rapid.Int64Range(minUnix, maxUnix).Draw(t, fmt.Sprintf("unix_%d", i)), 0).UTC()

			// Random hostname: safe alphanumeric labels + TLD (no HTML-escapable chars)
			label1 := rapid.StringMatching(`[a-z][a-z0-9]{2,8}`).Draw(t, fmt.Sprintf("label1_%d", i))
			label2 := rapid.StringMatching(`[a-z][a-z0-9]{2,8}`).Draw(t, fmt.Sprintf("label2_%d", i))
			tld := rapid.SampledFrom([]string{"com", "net", "org", "io", "dev"}).Draw(t, fmt.Sprintf("tld_%d", i))
			hostname := label1 + "." + label2 + "." + tld

			// Random query type from realistic set
			qt := rapid.SampledFrom(queryTypes).Draw(t, fmt.Sprintf("qt_%d", i))

			// Random client IP (X.X.X.X)
			ip := fmt.Sprintf("%d.%d.%d.%d",
				rapid.IntRange(1, 254).Draw(t, fmt.Sprintf("ip1_%d", i)),
				rapid.IntRange(0, 255).Draw(t, fmt.Sprintf("ip2_%d", i)),
				rapid.IntRange(0, 255).Draw(t, fmt.Sprintf("ip3_%d", i)),
				rapid.IntRange(1, 254).Draw(t, fmt.Sprintf("ip4_%d", i)),
			)

			isFiltered := rapid.Bool().Draw(t, fmt.Sprintf("filtered_%d", i))
			cached := rapid.Bool().Draw(t, fmt.Sprintf("cached_%d", i))

			entries[i] = LogEntry{
				Timestamp:  ts,
				Hostname:   hostname,
				QueryType:  qt,
				ClientIP:   ip,
				IsFiltered: isFiltered,
				Cached:     cached,
			}
		}

		// Create TemplateData with a single page containing all entries
		data := TemplateData{
			Page: Page{
				Entries:    entries,
				PageNum:    1,
				TotalPages: 1,
			},
		}

		// Render
		var buf bytes.Buffer
		if err := RenderPage(&buf, tmpl, data); err != nil {
			t.Fatalf("RenderPage: %v", err)
		}
		out := buf.String()

		// Verify each entry's fields appear in the rendered output
		for i, entry := range entries {
			wantTimestamp := entry.Timestamp.Format("2006-01-02 15:04:05")
			if !strings.Contains(out, wantTimestamp) {
				t.Fatalf("entry[%d]: output missing timestamp %q", i, wantTimestamp)
			}
			if !strings.Contains(out, entry.Hostname) {
				t.Fatalf("entry[%d]: output missing hostname %q", i, entry.Hostname)
			}
			if !strings.Contains(out, entry.QueryType) {
				t.Fatalf("entry[%d]: output missing query type %q", i, entry.QueryType)
			}
			if !strings.Contains(out, entry.ClientIP) {
				t.Fatalf("entry[%d]: output missing client IP %q", i, entry.ClientIP)
			}

			// Filter status
			if entry.IsFiltered {
				if !strings.Contains(out, "Blocked") {
					t.Fatalf("entry[%d]: output missing 'Blocked' for filtered entry", i)
				}
			} else {
				if !strings.Contains(out, "Allowed") {
					t.Fatalf("entry[%d]: output missing 'Allowed' for non-filtered entry", i)
				}
			}

			// Cached status
			if entry.Cached {
				if !strings.Contains(out, "Yes") {
					t.Fatalf("entry[%d]: output missing 'Yes' for cached entry", i)
				}
			} else {
				if !strings.Contains(out, "No") {
					t.Fatalf("entry[%d]: output missing 'No' for non-cached entry", i)
				}
			}
		}
	})
}

// Feature: adguard-log-viewer, Property 6: Blocked rows are visually distinguished
// **Validates: Requirements 3.4**
//
// For any LogEntry where IsFiltered is true, the rendered HTML for that row
// should contain a CSS class or element that distinguishes it from non-blocked rows.
func TestProperty_BlockedRowDistinction(t *testing.T) {
	tmpl, err := LoadTemplate("template.html")
	if err != nil {
		t.Fatalf("LoadTemplate: %v", err)
	}

	minUnix := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC).Unix()
	maxUnix := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC).Unix()
	queryTypes := []string{"A", "AAAA", "HTTPS", "SRV", "MX", "TXT", "CNAME"}

	rapid.Check(t, func(t *rapid.T) {
		n := rapid.IntRange(1, 15).Draw(t, "numEntries")
		entries := make([]LogEntry, n)
		hostnames := make([]string, n)
		filtered := make([]bool, n)

		for i := 0; i < n; i++ {
			ts := time.Unix(rapid.Int64Range(minUnix, maxUnix).Draw(t, fmt.Sprintf("unix_%d", i)), 0).UTC()

			// Unique hostname per entry so we can locate its row in the output
			label := rapid.StringMatching(`[a-z][a-z0-9]{2,8}`).Draw(t, fmt.Sprintf("label_%d", i))
			tld := rapid.SampledFrom([]string{"com", "net", "org", "io"}).Draw(t, fmt.Sprintf("tld_%d", i))
			hostname := fmt.Sprintf("host-%d-%s.%s", i, label, tld)

			qt := rapid.SampledFrom(queryTypes).Draw(t, fmt.Sprintf("qt_%d", i))
			ip := fmt.Sprintf("%d.%d.%d.%d",
				rapid.IntRange(1, 254).Draw(t, fmt.Sprintf("ip1_%d", i)),
				rapid.IntRange(0, 255).Draw(t, fmt.Sprintf("ip2_%d", i)),
				rapid.IntRange(0, 255).Draw(t, fmt.Sprintf("ip3_%d", i)),
				rapid.IntRange(1, 254).Draw(t, fmt.Sprintf("ip4_%d", i)),
			)
			isFiltered := rapid.Bool().Draw(t, fmt.Sprintf("filtered_%d", i))
			cached := rapid.Bool().Draw(t, fmt.Sprintf("cached_%d", i))

			entries[i] = LogEntry{
				Timestamp:  ts,
				Hostname:   hostname,
				QueryType:  qt,
				ClientIP:   ip,
				IsFiltered: isFiltered,
				Cached:     cached,
			}
			hostnames[i] = hostname
			filtered[i] = isFiltered
		}

		data := TemplateData{
			Page: Page{
				Entries:    entries,
				PageNum:    1,
				TotalPages: 1,
			},
		}

		var buf bytes.Buffer
		if err := RenderPage(&buf, tmpl, data); err != nil {
			t.Fatalf("RenderPage: %v", err)
		}

		lines := strings.Split(buf.String(), "\n")

		for i, hostname := range hostnames {
			// Find the <tr> line containing this entry's unique hostname
			var row string
			for _, line := range lines {
				if strings.Contains(line, hostname) {
					row = line
					break
				}
			}
			if row == "" {
				t.Fatalf("entry[%d]: could not find row for hostname %q", i, hostname)
			}

			hasBlockedClass := strings.Contains(row, `class="blocked"`)

			if filtered[i] && !hasBlockedClass {
				t.Fatalf("entry[%d]: IsFiltered=true but row missing class=\"blocked\" for %q", i, hostname)
			}
			if !filtered[i] && hasBlockedClass {
				t.Fatalf("entry[%d]: IsFiltered=false but row has class=\"blocked\" for %q", i, hostname)
			}
		}
	})
}

// Feature: adguard-log-viewer, Property 13: Filter values preserved in rendered form
// **Validates: Requirements 8.2**
//
// For any set of active filter parameters, the rendered HTML page should contain
// form input elements whose values match the submitted filter parameters.
func TestProperty_FilterValuesPreserved(t *testing.T) {
	tmpl, err := LoadTemplate("template.html")
	if err != nil {
		t.Fatalf("LoadTemplate: %v", err)
	}

	minUnix := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC).Unix()
	maxUnix := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC).Unix()

	rapid.Check(t, func(t *rapid.T) {
		// Generate random filter params with safe characters only
		ip := rapid.SampledFrom([]string{
			"",
			rapid.StringMatching(`[a-z0-9]{3,10}`).Draw(t, "ipVal"),
		}).Draw(t, "ip")

		hostname := rapid.SampledFrom([]string{
			"",
			rapid.StringMatching(`[a-z]{3,8}`).Draw(t, "hostnameVal"),
		}).Draw(t, "hostname")

		blockStatus := rapid.SampledFrom([]string{"blocked", "allowed", ""}).Draw(t, "blockStatus")

		// Generate optional TimeStart
		hasStart := rapid.Bool().Draw(t, "hasStart")
		var timeStart *time.Time
		var startUnix int64
		if hasStart {
			startUnix = rapid.Int64Range(minUnix, maxUnix).Draw(t, "startUnix")
			ts := time.Unix(startUnix, 0).UTC()
			// Truncate to minute precision to match datetimeLocalFormat
			ts = ts.Truncate(time.Minute)
			timeStart = &ts
		}

		// Generate optional TimeEnd (>= TimeStart if both set)
		hasEnd := rapid.Bool().Draw(t, "hasEnd")
		var timeEnd *time.Time
		if hasEnd {
			var endUnix int64
			if hasStart {
				endUnix = rapid.Int64Range(startUnix, maxUnix).Draw(t, "endUnix")
			} else {
				endUnix = rapid.Int64Range(minUnix, maxUnix).Draw(t, "endUnix")
			}
			te := time.Unix(endUnix, 0).UTC()
			te = te.Truncate(time.Minute)
			timeEnd = &te
		}

		filter := FilterParams{
			IP:          ip,
			Hostname:    hostname,
			TimeStart:   timeStart,
			TimeEnd:     timeEnd,
			BlockStatus: blockStatus,
		}

		data := TemplateData{
			Page: Page{
				Entries:    nil,
				PageNum:    1,
				TotalPages: 1,
			},
			Filter: filter,
		}

		var buf bytes.Buffer
		if err := RenderPage(&buf, tmpl, data); err != nil {
			t.Fatalf("RenderPage: %v", err)
		}
		out := buf.String()

		// Verify IP value preserved
		if ip != "" {
			want := fmt.Sprintf(`value="%s"`, ip)
			if !strings.Contains(out, want) {
				t.Fatalf("IP filter: output missing %s", want)
			}
		}

		// Verify Hostname value preserved
		if hostname != "" {
			want := fmt.Sprintf(`value="%s"`, hostname)
			if !strings.Contains(out, want) {
				t.Fatalf("Hostname filter: output missing %s", want)
			}
		}

		// Verify TimeStart value preserved
		if timeStart != nil {
			want := fmt.Sprintf(`value="%s"`, timeStart.Format(datetimeLocalFormat))
			if !strings.Contains(out, want) {
				t.Fatalf("TimeStart filter: output missing %s", want)
			}
		}

		// Verify TimeEnd value preserved
		if timeEnd != nil {
			want := fmt.Sprintf(`value="%s"`, timeEnd.Format(datetimeLocalFormat))
			if !strings.Contains(out, want) {
				t.Fatalf("TimeEnd filter: output missing %s", want)
			}
		}

		// Verify BlockStatus preserved in select
		switch blockStatus {
		case "blocked":
			if !strings.Contains(out, `"blocked" selected`) {
				t.Fatalf("BlockStatus=blocked: output missing selected attribute")
			}
		case "allowed":
			if !strings.Contains(out, `"allowed" selected`) {
				t.Fatalf("BlockStatus=allowed: output missing selected attribute")
			}
		case "":
			if strings.Contains(out, `"blocked" selected`) {
				t.Fatalf("BlockStatus empty: output should not have blocked selected")
			}
			if strings.Contains(out, `"allowed" selected`) {
				t.Fatalf("BlockStatus empty: output should not have allowed selected")
			}
		}
	})
}

// Feature: adguard-log-viewer, Property 14: HTML page size under 64KB
// **Validates: Requirements 9.3**
//
// For any single page of rendered results (up to the configured page size),
// the total HTML output should be less than 65,536 bytes.
func TestProperty_HTMLPageSizeUnder64KB(t *testing.T) {
	tmpl, err := LoadTemplate("template.html")
	if err != nil {
		t.Fatalf("LoadTemplate: %v", err)
	}

	minUnix := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC).Unix()
	maxUnix := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC).Unix()

	queryTypes := []string{"A", "AAAA", "HTTPS", "SRV", "MX", "TXT", "CNAME"}
	tlds := []string{"com", "net", "org", "io", "dev"}

	rapid.Check(t, func(t *rapid.T) {
		// Generate 0 to defaultPageSize (50) entries — the max a single page can hold
		n := rapid.IntRange(0, defaultPageSize).Draw(t, "numEntries")
		entries := make([]LogEntry, n)

		for i := 0; i < n; i++ {
			ts := time.Unix(rapid.Int64Range(minUnix, maxUnix).Draw(t, fmt.Sprintf("unix_%d", i)), 0).UTC()

			// Realistic domain name up to ~30 chars: label.label.tld
			label1 := rapid.StringMatching(`[a-z][a-z0-9]{2,10}`).Draw(t, fmt.Sprintf("l1_%d", i))
			label2 := rapid.StringMatching(`[a-z][a-z0-9]{2,10}`).Draw(t, fmt.Sprintf("l2_%d", i))
			tld := rapid.SampledFrom(tlds).Draw(t, fmt.Sprintf("tld_%d", i))
			hostname := label1 + "." + label2 + "." + tld

			qt := rapid.SampledFrom(queryTypes).Draw(t, fmt.Sprintf("qt_%d", i))

			ip := fmt.Sprintf("%d.%d.%d.%d",
				rapid.IntRange(1, 254).Draw(t, fmt.Sprintf("ip1_%d", i)),
				rapid.IntRange(0, 255).Draw(t, fmt.Sprintf("ip2_%d", i)),
				rapid.IntRange(0, 255).Draw(t, fmt.Sprintf("ip3_%d", i)),
				rapid.IntRange(1, 254).Draw(t, fmt.Sprintf("ip4_%d", i)),
			)

			entries[i] = LogEntry{
				Timestamp:  ts,
				Hostname:   hostname,
				QueryType:  qt,
				ClientIP:   ip,
				IsFiltered: rapid.Bool().Draw(t, fmt.Sprintf("filt_%d", i)),
				Cached:     rapid.Bool().Draw(t, fmt.Sprintf("cache_%d", i)),
			}
		}

		// Random filter params to exercise the form rendering too
		filter := FilterParams{
			IP:          rapid.StringMatching(`[0-9]{0,3}`).Draw(t, "filterIP"),
			Hostname:    rapid.StringMatching(`[a-z]{0,8}`).Draw(t, "filterHost"),
			BlockStatus: rapid.SampledFrom([]string{"blocked", "allowed", ""}).Draw(t, "filterStatus"),
		}

		totalPages := 1
		if n > 0 {
			totalPages = 1
		}

		data := TemplateData{
			Page: Page{
				Entries:    entries,
				PageNum:    1,
				TotalPages: totalPages,
				HasPrev:    false,
				HasNext:    false,
			},
			Filter: filter,
		}

		var buf bytes.Buffer
		if err := RenderPage(&buf, tmpl, data); err != nil {
			t.Fatalf("RenderPage: %v", err)
		}

		if buf.Len() >= 65536 {
			t.Fatalf("rendered HTML is %d bytes, expected < 65536", buf.Len())
		}
	})
}
