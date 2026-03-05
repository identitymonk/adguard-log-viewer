package main

import (
	"fmt"
	"html/template"
	"net/http"
	"strconv"
)

const defaultPageSize = 50

// NewHandler returns an http.HandlerFunc that serves the log viewer page.
func NewHandler(cfg Config, tmpl *template.Template) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		filterParams := ParseFilterParams(r.URL.Query())

		pageNum := 1
		if p := r.URL.Query().Get("page"); p != "" {
			if n, err := strconv.Atoi(p); err == nil && n >= 1 {
				pageNum = n
			}
		}

		filter := BuildFilter(filterParams)

		entries, err := ParseLogFile(cfg.LogFilePath, filter)
		if err != nil {
			data := TemplateData{
				Error:  fmt.Sprintf("Error reading log file: %s", err),
				Filter: filterParams,
			}
			if renderErr := RenderPage(w, tmpl, data); renderErr != nil {
				http.Error(w, "Internal server error", http.StatusInternalServerError)
			}
			return
		}

		page := Paginate(entries, pageNum, defaultPageSize)

		data := TemplateData{
			Page:   page,
			Filter: filterParams,
		}
		if err := RenderPage(w, tmpl, data); err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
	}
}
