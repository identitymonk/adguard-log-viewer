package main

import (
	"fmt"
	"html/template"
	"io"
	"net/url"
)

// TemplateData holds all data passed to the HTML template.
type TemplateData struct {
	Page   Page
	Filter FilterParams
	Error  string
}

// TemplateFuncs returns the custom FuncMap used by template.html.
func TemplateFuncs() template.FuncMap {
	return template.FuncMap{
		"add": func(a, b int) int { return a + b },
		"sub": func(a, b int) int { return a - b },
		"filterQuery": func(f FilterParams, page int) template.URL {
			v := url.Values{}
			if f.IP != "" {
				v.Set("ip", f.IP)
			}
			if f.Hostname != "" {
				v.Set("hostname", f.Hostname)
			}
			if f.TimeStart != nil {
				v.Set("start", f.TimeStart.Format(datetimeLocalFormat))
			}
			if f.TimeEnd != nil {
				v.Set("end", f.TimeEnd.Format(datetimeLocalFormat))
			}
			if f.BlockStatus != "" {
				v.Set("status", f.BlockStatus)
			}
			v.Set("page", fmt.Sprintf("%d", page))
			return template.URL(v.Encode())
		},
	}
}

// LoadTemplate parses the HTML template file at path with custom functions.
func LoadTemplate(path string) (*template.Template, error) {
	return template.New("template.html").Funcs(TemplateFuncs()).ParseFiles(path)
}

// RenderPage executes the template with the given data, writing to w.
func RenderPage(w io.Writer, tmpl *template.Template, data TemplateData) error {
	return tmpl.Execute(w, data)
}
