package main

import (
	"fmt"
	"net/http"
	"os"
)

func main() {
	configPath := "config.txt"
	if len(os.Args) > 1 {
		configPath = os.Args[1]
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %s\n", err)
		os.Exit(1)
	}

	tmpl, err := LoadTemplate("template.html")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading template: %s\n", err)
		os.Exit(1)
	}

	handler := NewHandler(cfg, tmpl)
	http.HandleFunc("/", handler)

	fmt.Printf("Listening on :%d\n", cfg.HTTPPort)
	if err := http.ListenAndServe(fmt.Sprintf(":%d", cfg.HTTPPort), nil); err != nil {
		fmt.Fprintf(os.Stderr, "Error starting server: %s\n", err)
		os.Exit(1)
	}
}
