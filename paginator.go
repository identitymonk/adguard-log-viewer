package main

// Page holds a single page of log entries along with pagination metadata.
type Page struct {
	Entries    []LogEntry
	PageNum    int
	TotalPages int
	HasPrev    bool
	HasNext    bool
}

// Paginate takes entries in file order (oldest first), reverses them so newest
// entries come first, and returns the requested page slice.
func Paginate(entries []LogEntry, pageNum int, pageSize int) Page {
	if pageSize < 1 {
		pageSize = 1
	}

	n := len(entries)

	// Reverse into a new slice (don't mutate the original).
	reversed := make([]LogEntry, n)
	for i, e := range entries {
		reversed[n-1-i] = e
	}

	// Compute total pages (ceil division). Empty list → 1 page.
	totalPages := 1
	if n > 0 {
		totalPages = (n + pageSize - 1) / pageSize
	}

	// Clamp page number.
	if pageNum < 1 {
		pageNum = 1
	}
	if pageNum > totalPages {
		pageNum = totalPages
	}

	// Slice for the requested page.
	start := (pageNum - 1) * pageSize
	end := start + pageSize
	if end > n {
		end = n
	}

	var pageEntries []LogEntry
	if start < n {
		pageEntries = reversed[start:end]
	} else {
		pageEntries = []LogEntry{}
	}

	return Page{
		Entries:    pageEntries,
		PageNum:    pageNum,
		TotalPages: totalPages,
		HasPrev:    pageNum > 1,
		HasNext:    pageNum < totalPages,
	}
}
