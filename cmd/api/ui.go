package main

import (
	"fmt"
	"html/template"
	"net/http"
)

func (app *application) home(w http.ResponseWriter, r *http.Request) {
	ts, err := template.ParseFiles("./ui/html/pages/home.tmpl")
	if err != nil {
		app.logger.Error(err.Error())
		return
	}
	
	err = ts.Execute(w, nil)
	if err != nil {
		app.logger.Error(err.Error())
	}
}

func (app *application) imageView(w http.ResponseWriter, r *http.Request) {
	id, err := app.readIDParam(r)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	fmt.Fprintf(w, "display a specific image with ID %d...", id)
}

func (app *application) imageStore(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("display a form for storing a new image..."))
}
