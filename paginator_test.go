package main

import (
	"testing"
	"time"
)

// helper to create a LogEntry with a distinct timestamp based on day number.
func entry(day int) LogEntry {
	return LogEntry{
		Timestamp: time.Date(2024, 1, day, 12, 0, 0, 0, time.UTC),
		Hostname:  "example.com",
	}
}

func TestPaginate_EmptyList(t *testing.T) {
	page := Paginate(nil, 1, 10)

	if len(page.Entries) != 0 {
		t.Errorf("Entries length = %d, want 0", len(page.Entries))
	}
	if page.PageNum != 1 {
		t.Errorf("PageNum = %d, want 1", page.PageNum)
	}
	if page.TotalPages != 1 {
		t.Errorf("TotalPages = %d, want 1", page.TotalPages)
	}
	if page.HasPrev {
		t.Error("HasPrev = true, want false")
	}
	if page.HasNext {
		t.Error("HasNext = true, want false")
	}
}

func TestPaginate_SinglePage(t *testing.T) {
	entries := []LogEntry{entry(1), entry(2), entry(3)}
	page := Paginate(entries, 1, 10)

	if len(page.Entries) != 3 {
		t.Fatalf("Entries length = %d, want 3", len(page.Entries))
	}
	// Entries should be reversed: day 3, day 2, day 1
	wantDays := []int{3, 2, 1}
	for i, want := range wantDays {
		got := page.Entries[i].Timestamp.Day()
		if got != want {
			t.Errorf("Entries[%d] day = %d, want %d", i, got, want)
		}
	}
	if page.PageNum != 1 {
		t.Errorf("PageNum = %d, want 1", page.PageNum)
	}
	if page.TotalPages != 1 {
		t.Errorf("TotalPages = %d, want 1", page.TotalPages)
	}
	if page.HasPrev {
		t.Error("HasPrev = true, want false")
	}
	if page.HasNext {
		t.Error("HasNext = true, want false")
	}
}

func TestPaginate_MultiplePages(t *testing.T) {
	// 5 entries in file order (oldest first): day 1..5
	entries := []LogEntry{entry(1), entry(2), entry(3), entry(4), entry(5)}

	// Page 1: newest 2 entries (day 5, day 4)
	p1 := Paginate(entries, 1, 2)
	if len(p1.Entries) != 2 {
		t.Fatalf("Page 1 entries = %d, want 2", len(p1.Entries))
	}
	if p1.Entries[0].Timestamp.Day() != 5 || p1.Entries[1].Timestamp.Day() != 4 {
		t.Errorf("Page 1 days = [%d, %d], want [5, 4]",
			p1.Entries[0].Timestamp.Day(), p1.Entries[1].Timestamp.Day())
	}
	if p1.PageNum != 1 {
		t.Errorf("Page 1 PageNum = %d, want 1", p1.PageNum)
	}
	if p1.TotalPages != 3 {
		t.Errorf("Page 1 TotalPages = %d, want 3", p1.TotalPages)
	}
	if p1.HasPrev {
		t.Error("Page 1 HasPrev = true, want false")
	}
	if !p1.HasNext {
		t.Error("Page 1 HasNext = false, want true")
	}

	// Page 2: day 3, day 2
	p2 := Paginate(entries, 2, 2)
	if len(p2.Entries) != 2 {
		t.Fatalf("Page 2 entries = %d, want 2", len(p2.Entries))
	}
	if p2.Entries[0].Timestamp.Day() != 3 || p2.Entries[1].Timestamp.Day() != 2 {
		t.Errorf("Page 2 days = [%d, %d], want [3, 2]",
			p2.Entries[0].Timestamp.Day(), p2.Entries[1].Timestamp.Day())
	}
	if !p2.HasPrev {
		t.Error("Page 2 HasPrev = false, want true")
	}
	if !p2.HasNext {
		t.Error("Page 2 HasNext = false, want true")
	}

	// Page 3: day 1 (last entry)
	p3 := Paginate(entries, 3, 2)
	if len(p3.Entries) != 1 {
		t.Fatalf("Page 3 entries = %d, want 1", len(p3.Entries))
	}
	if p3.Entries[0].Timestamp.Day() != 1 {
		t.Errorf("Page 3 day = %d, want 1", p3.Entries[0].Timestamp.Day())
	}
	if !p3.HasPrev {
		t.Error("Page 3 HasPrev = false, want true")
	}
	if p3.HasNext {
		t.Error("Page 3 HasNext = true, want false")
	}
}

func TestPaginate_OutOfRange_TooHigh(t *testing.T) {
	entries := []LogEntry{entry(1), entry(2), entry(3), entry(4), entry(5)}
	page := Paginate(entries, 99, 2)

	// Should clamp to last page (page 3)
	if page.PageNum != 3 {
		t.Errorf("PageNum = %d, want 3 (clamped)", page.PageNum)
	}
	if len(page.Entries) != 1 {
		t.Fatalf("Entries length = %d, want 1", len(page.Entries))
	}
	if page.Entries[0].Timestamp.Day() != 1 {
		t.Errorf("Entry day = %d, want 1 (oldest)", page.Entries[0].Timestamp.Day())
	}
	if !page.HasPrev {
		t.Error("HasPrev = false, want true")
	}
	if page.HasNext {
		t.Error("HasNext = true, want false")
	}
}

func TestPaginate_OutOfRange_TooLow(t *testing.T) {
	entries := []LogEntry{entry(1), entry(2), entry(3)}
	page := Paginate(entries, 0, 10)

	// Should clamp to page 1
	if page.PageNum != 1 {
		t.Errorf("PageNum = %d, want 1 (clamped)", page.PageNum)
	}
	if len(page.Entries) != 3 {
		t.Fatalf("Entries length = %d, want 3", len(page.Entries))
	}
	// Reversed: day 3, 2, 1
	if page.Entries[0].Timestamp.Day() != 3 {
		t.Errorf("First entry day = %d, want 3", page.Entries[0].Timestamp.Day())
	}
	if page.HasPrev {
		t.Error("HasPrev = true, want false")
	}
	if page.HasNext {
		t.Error("HasNext = true, want false")
	}
}

func TestPaginate_PageSizeOne(t *testing.T) {
	entries := []LogEntry{entry(1), entry(2), entry(3)}

	// 3 entries, pageSize=1 → 3 pages, each with 1 entry in reverse order
	for pageNum := 1; pageNum <= 3; pageNum++ {
		page := Paginate(entries, pageNum, 1)

		if page.TotalPages != 3 {
			t.Errorf("Page %d: TotalPages = %d, want 3", pageNum, page.TotalPages)
		}
		if len(page.Entries) != 1 {
			t.Fatalf("Page %d: entries = %d, want 1", pageNum, len(page.Entries))
		}

		// Reversed order: page 1→day 3, page 2→day 2, page 3→day 1
		wantDay := 4 - pageNum
		if page.Entries[0].Timestamp.Day() != wantDay {
			t.Errorf("Page %d: day = %d, want %d", pageNum, page.Entries[0].Timestamp.Day(), wantDay)
		}

		wantHasPrev := pageNum > 1
		wantHasNext := pageNum < 3
		if page.HasPrev != wantHasPrev {
			t.Errorf("Page %d: HasPrev = %v, want %v", pageNum, page.HasPrev, wantHasPrev)
		}
		if page.HasNext != wantHasNext {
			t.Errorf("Page %d: HasNext = %v, want %v", pageNum, page.HasNext, wantHasNext)
		}
	}
}
