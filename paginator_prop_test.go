package main

import (
	"sort"
	"testing"
	"time"

	"pgregory.net/rapid"
)

// Feature: adguard-log-viewer, Property 5: Descending chronological order
// **Validates: Requirements 3.2**
//
// For any list of LogEntry values with distinct timestamps, after pagination
// the entries on each page should be in strictly descending timestamp order.
func TestProperty_DescendingOrder(t *testing.T) {
	// Time range: 2020-01-01 to 2026-01-01 in Unix seconds
	minUnix := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC).Unix()
	maxUnix := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC).Unix()

	rapid.Check(t, func(t *rapid.T) {
		// Generate a slice of unique Unix seconds (distinct timestamps)
		n := rapid.IntRange(1, 100).Draw(t, "numEntries")
		unixSet := make(map[int64]bool, n)
		unixSlice := make([]int64, 0, n)
		for len(unixSlice) < n {
			ts := rapid.Int64Range(minUnix, maxUnix).Draw(t, "unix")
			if !unixSet[ts] {
				unixSet[ts] = true
				unixSlice = append(unixSlice, ts)
			}
		}

		// Sort ascending to simulate file order (oldest first)
		sort.Slice(unixSlice, func(i, j int) bool {
			return unixSlice[i] < unixSlice[j]
		})

		// Create LogEntry values from the sorted timestamps
		entries := make([]LogEntry, n)
		for i, unix := range unixSlice {
			entries[i] = LogEntry{
				Timestamp: time.Unix(unix, 0).UTC(),
				Hostname:  "example.com",
			}
		}

		// Generate a random page size between 1 and len(entries)+1
		pageSize := rapid.IntRange(1, n+1).Draw(t, "pageSize")

		// Paginate and get total pages
		firstPage := Paginate(entries, 1, pageSize)
		totalPages := firstPage.TotalPages

		// Verify each page is in strictly descending timestamp order
		for pageNum := 1; pageNum <= totalPages; pageNum++ {
			page := Paginate(entries, pageNum, pageSize)

			for i := 1; i < len(page.Entries); i++ {
				prev := page.Entries[i-1].Timestamp
				curr := page.Entries[i].Timestamp
				if !prev.After(curr) {
					t.Fatalf("page %d: entry[%d].Timestamp (%v) is not strictly after entry[%d].Timestamp (%v)",
						pageNum, i-1, prev, i, curr)
				}
			}
		}

		// Verify descending order across consecutive pages:
		// last entry of page N should be strictly after first entry of page N+1
		for pageNum := 1; pageNum < totalPages; pageNum++ {
			currPage := Paginate(entries, pageNum, pageSize)
			nextPage := Paginate(entries, pageNum+1, pageSize)

			if len(currPage.Entries) == 0 || len(nextPage.Entries) == 0 {
				continue
			}

			lastOfCurr := currPage.Entries[len(currPage.Entries)-1].Timestamp
			firstOfNext := nextPage.Entries[0].Timestamp

			if !lastOfCurr.After(firstOfNext) {
				t.Fatalf("cross-page order violation: page %d last entry (%v) is not strictly after page %d first entry (%v)",
					pageNum, lastOfCurr, pageNum+1, firstOfNext)
			}
		}
	})
}

// Feature: adguard-log-viewer, Property 7: Page size limit
// **Validates: Requirements 3.5**
//
// For any list of LogEntry values and any page size n > 0, each page returned
// by Paginate should contain at most n entries.
func TestProperty_PageSizeLimit(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate a random number of entries (0 to 200)
		n := rapid.IntRange(0, 200).Draw(t, "numEntries")

		// Build entries with arbitrary timestamps
		entries := make([]LogEntry, n)
		for i := 0; i < n; i++ {
			entries[i] = LogEntry{
				Timestamp: time.Unix(rapid.Int64Range(0, 2000000000).Draw(t, "unix"), 0).UTC(),
				Hostname:  "example.com",
			}
		}

		// Generate a random page size (1 to 50)
		pageSize := rapid.IntRange(1, 50).Draw(t, "pageSize")

		// Get total pages from the first call
		firstPage := Paginate(entries, 1, pageSize)
		totalPages := firstPage.TotalPages

		// Verify each page has at most pageSize entries
		for pageNum := 1; pageNum <= totalPages; pageNum++ {
			page := Paginate(entries, pageNum, pageSize)

			if len(page.Entries) > pageSize {
				t.Fatalf("page %d: got %d entries, want at most %d",
					pageNum, len(page.Entries), pageSize)
			}
		}

		// Verify all pages except possibly the last have exactly pageSize entries
		if n > 0 {
			for pageNum := 1; pageNum < totalPages; pageNum++ {
				page := Paginate(entries, pageNum, pageSize)

				if len(page.Entries) != pageSize {
					t.Fatalf("page %d (not last): got %d entries, want exactly %d",
						pageNum, len(page.Entries), pageSize)
				}
			}
		}
	})
}
