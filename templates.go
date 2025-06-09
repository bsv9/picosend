package main

import (
	"html/template"
	"net/http"
)


func homeHandler(w http.ResponseWriter, r *http.Request) {
	tmpl := template.Must(template.ParseFS(templatesFS, "templates/home.html"))
	tmpl.Execute(w, nil)
}

func viewSecretHandler(w http.ResponseWriter, r *http.Request) {
	tmpl := template.Must(template.ParseFS(templatesFS, "templates/view-secret.html"))
	tmpl.Execute(w, nil)
}