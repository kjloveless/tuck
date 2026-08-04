package main

import (
	"errors"
	"html/template"
	"net/http"

	"tuck.loveless.dev/internal/data"
)

func (app *application) home(w http.ResponseWriter, r *http.Request) {
	files := []string{
		"./ui/html/base.tmpl",
		"./ui/html/partials/nav.tmpl",
		"./ui/html/pages/home.tmpl",
	}

	ts, err := template.ParseFiles(files...)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	err = ts.ExecuteTemplate(w, "base", nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *application) imageView(w http.ResponseWriter, r *http.Request) {
	id, err := app.readIDParam(r)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	image, err := app.models.Images.Get(id)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			app.notFoundResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	files := []string{
		"./ui/html/base.tmpl",
		"./ui/html/partials/nav.tmpl",
		"./ui/html/pages/view.tmpl",
	}

	ts, err := template.ParseFiles(files...)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	err = ts.ExecuteTemplate(w, "base", image)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *application) imageStore(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("display a form for storing a new image..."))
}
