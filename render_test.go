package main

import (
	"bytes"
	"html/template"
	"strings"
	"testing"
	"time"
)

// testTemplate loads the template once for reuse across tests.
func testTemplate(t *testing.T) *template.Template {
	t.Helper()
	tmpl, err := LoadTemplate("template.html")
	if err != nil {
		t.Fatalf("LoadTemplate: %v", err)
	}
	return tmpl
}

func TestRender_ColumnValues(t *testing.T) {
	tmpl := testTemplate(t)

	ts := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	data := TemplateData{
		Page: Page{
			Entries: []LogEntry{
				{
					Timestamp:  ts,
					Hostname:   "example.com",
					QueryType:  "A",
					ClientIP:   "192.168.1.1",
					IsFiltered: false,
					Cached:     true,
				},
				{
					Timestamp:  ts,
					Hostname:   "blocked.org",
					QueryType:  "AAAA",
					ClientIP:   "10.0.0.5",
					IsFiltered: true,
					Cached:     false,
				},
			},
			PageNum:    1,
			TotalPages: 1,
		},
	}

	var buf bytes.Buffer
	if err := RenderPage(&buf, tmpl, data); err != nil {
		t.Fatalf("RenderPage: %v", err)
	}
	out := buf.String()

	checks := []string{
		"2024-01-15 10:30:00",
		"example.com",
		"A",
		"192.168.1.1",
		"Allowed",
		"Yes",
		"blocked.org",
		"AAAA",
		"10.0.0.5",
		"Blocked",
		"No",
	}
	for _, want := range checks {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q", want)
		}
	}
}

func TestRender_BlockedRowClass(t *testing.T) {
	tmpl := testTemplate(t)

	ts := time.Date(2024, 6, 1, 12, 0, 0, 0, time.UTC)
	data := TemplateData{
		Page: Page{
			Entries: []LogEntry{
				{Timestamp: ts, Hostname: "allowed.com", ClientIP: "1.1.1.1", IsFiltered: false},
				{Timestamp: ts, Hostname: "blocked.com", ClientIP: "2.2.2.2", IsFiltered: true},
			},
			PageNum:    1,
			TotalPages: 1,
		},
	}

	var buf bytes.Buffer
	if err := RenderPage(&buf, tmpl, data); err != nil {
		t.Fatalf("RenderPage: %v", err)
	}
	out := buf.String()

	// Find all <tr> tags that contain entry data (skip the header row).
	// The blocked row should have class="blocked", the allowed row should not.
	lines := strings.Split(out, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "<tr") || strings.Contains(trimmed, "<th>") {
			continue
		}
		if strings.Contains(trimmed, "blocked.com") {
			if !strings.Contains(trimmed, `class="blocked"`) {
				t.Error("blocked row missing class=\"blocked\"")
			}
		}
		if strings.Contains(trimmed, "allowed.com") {
			if strings.Contains(trimmed, `class="blocked"`) {
				t.Error("allowed row should not have class=\"blocked\"")
			}
		}
	}
}

func TestRender_FilterFormPreservesValues(t *testing.T) {
	tmpl := testTemplate(t)

	start := time.Date(2024, 3, 1, 8, 0, 0, 0, time.UTC)
	end := time.Date(2024, 3, 15, 18, 30, 0, 0, time.UTC)
	data := TemplateData{
		Page: Page{
			Entries:    []LogEntry{{Timestamp: start, Hostname: "x.com", ClientIP: "1.2.3.4"}},
			PageNum:    1,
			TotalPages: 1,
		},
		Filter: FilterParams{
			IP:          "192.168",
			Hostname:    "example",
			TimeStart:   &start,
			TimeEnd:     &end,
			BlockStatus: "blocked",
		},
	}

	var buf bytes.Buffer
	if err := RenderPage(&buf, tmpl, data); err != nil {
		t.Fatalf("RenderPage: %v", err)
	}
	out := buf.String()

	checks := []struct {
		desc string
		want string
	}{
		{"IP value", `value="192.168"`},
		{"hostname value", `value="example"`},
		{"start time", `value="2024-03-01T08:00"`},
		{"end time", `value="2024-03-15T18:30"`},
		{"blocked selected", `"blocked" selected`},
	}
	for _, c := range checks {
		if !strings.Contains(out, c.want) {
			t.Errorf("%s: output missing %q", c.desc, c.want)
		}
	}
}

func TestRender_PaginationLinksIncludeFilters(t *testing.T) {
	tmpl := testTemplate(t)

	start := time.Date(2024, 5, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2024, 5, 31, 23, 59, 0, 0, time.UTC)

	// Create enough entries for multiple pages (default page size doesn't matter;
	// we set TotalPages and HasNext directly).
	entries := make([]LogEntry, 3)
	for i := range entries {
		entries[i] = LogEntry{
			Timestamp: start,
			Hostname:  "test.com",
			ClientIP:  "10.0.0.1",
		}
	}

	data := TemplateData{
		Page: Page{
			Entries:    entries,
			PageNum:    1,
			TotalPages: 3,
			HasNext:    true,
			HasPrev:    false,
		},
		Filter: FilterParams{
			IP:          "10.0",
			Hostname:    "test",
			TimeStart:   &start,
			TimeEnd:     &end,
			BlockStatus: "allowed",
		},
	}

	var buf bytes.Buffer
	if err := RenderPage(&buf, tmpl, data); err != nil {
		t.Fatalf("RenderPage: %v", err)
	}
	out := buf.String()

	// The "Next" pagination link should contain all filter params.
	wantParams := []string{
		"ip=10.0",
		"hostname=test",
		"status=allowed",
		"start=",
		"end=",
		"page=2",
	}
	for _, p := range wantParams {
		if !strings.Contains(out, p) {
			t.Errorf("pagination link missing param %q", p)
		}
	}
}

func TestRender_EmptyEntries(t *testing.T) {
	tmpl := testTemplate(t)

	data := TemplateData{
		Page: Page{
			Entries:    nil,
			PageNum:    1,
			TotalPages: 1,
		},
	}

	var buf bytes.Buffer
	if err := RenderPage(&buf, tmpl, data); err != nil {
		t.Fatalf("RenderPage: %v", err)
	}
	out := buf.String()

	if !strings.Contains(out, "No log entries found.") {
		t.Error("expected 'No log entries found.' for empty entries")
	}
}

func TestRender_ErrorMessage(t *testing.T) {
	tmpl := testTemplate(t)

	data := TemplateData{
		Error: "could not read log file: /var/log/query.json",
	}

	var buf bytes.Buffer
	if err := RenderPage(&buf, tmpl, data); err != nil {
		t.Fatalf("RenderPage: %v", err)
	}
	out := buf.String()

	if !strings.Contains(out, "could not read log file: /var/log/query.json") {
		t.Error("expected error message in output")
	}
	if strings.Contains(out, "<table>") {
		t.Error("table should not be rendered when error is set")
	}
}
