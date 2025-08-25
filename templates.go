package main

import (
	"html/template"
	"net/http"
	"strings"
)

func homeHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	tmpl := template.Must(template.ParseFS(templatesFS, "templates/home.html"))
	tmpl.Execute(w, nil)
}

func viewSecretHandler(w http.ResponseWriter, r *http.Request) {
	// Build the base URL for Open Graph meta tags
	scheme := "https"
	if r.Header.Get("X-Forwarded-Proto") != "" {
		scheme = r.Header.Get("X-Forwarded-Proto")
	} else if r.TLS == nil && !strings.Contains(r.Host, "localhost") && !strings.Contains(r.Host, "127.0.0.1") {
		scheme = "http"
	}

	baseURL := scheme + "://" + r.Host
	requestURL := baseURL + r.URL.Path

	data := struct {
		BaseURL    string
		RequestURL string
	}{
		BaseURL:    baseURL,
		RequestURL: requestURL,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	tmpl := template.Must(template.ParseFS(templatesFS, "templates/view-secret.html"))
	tmpl.Execute(w, data)
}
