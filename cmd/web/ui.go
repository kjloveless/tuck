package main

import (
	"errors"
	"net/http"

	"tuck.loveless.dev/internal/data"
)

func (app *application) home(w http.ResponseWriter, r *http.Request) {
	images, _, err := app.models.Images.Latest()
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	data := app.newTemplateData(r)
	data.Images = images

	app.render(w, r, http.StatusOK, "home.tmpl",  data)
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

	data := app.newTemplateData(r)
	data.Image = image

	app.render(w, r, http.StatusOK, "view.tmpl", data)
}

func (app *application) imageStore(w http.ResponseWriter, r *http.Request) {
	data := app.newTemplateData(r)

	app.render(w, r, http.StatusOK, "store.tmpl", data)
}
