package main

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"tuck.loveless.dev/internal/data"
)

// /-----------------------------------------------------------------------------
func (app *application) home(w http.ResponseWriter, r *http.Request) {
	images, _, err := app.models.Images.Latest()
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	data := app.newTemplateData(r)
	data.Images = images

	app.render(w, r, http.StatusOK, "home.tmpl", data)
}

// /-----------------------------------------------------------------------------
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

// /-----------------------------------------------------------------------------
func (app *application) imageStore(w http.ResponseWriter, r *http.Request) {
	data := app.newTemplateData(r)

	app.render(w, r, http.StatusOK, "store.tmpl", data)
}

// /-----------------------------------------------------------------------------
func (app *application) imageStorePost(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	location := r.PostForm.Get("location")
	year, err := strconv.Atoi(r.PostForm.Get("year"))
	people := strings.Split(r.PostForm.Get("people"), "\r\n")
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	image := data.Image{
		Location: location,
		Year:     year,
		People:   people,
	}

	image, err = app.models.Images.Insert(image)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/images/%d", image.ID), http.StatusSeeOther)
}

// /-----------------------------------------------------------------------------
func (app *application) imageDelete(w http.ResponseWriter, r *http.Request) {
	id, err := app.readIDParam(r)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	err = app.models.Images.Delete(id)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			app.notFoundResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}
