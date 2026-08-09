package main

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"unicode/utf8"

	"tuck.loveless.dev/internal/data"
)

type imageStoreForm struct {
	Location    string
	Year        int
	People      []string
	FieldErrors map[string]string
}

type userSignupForm struct {
	Name        string `form:"name"`
	Email       string `form:"email"`
	Password    string `form:"password"`
	FieldErrors map[string]string
}

// ------------------------------------------------------------------------------
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

// ------------------------------------------------------------------------------
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

// ------------------------------------------------------------------------------
func (app *application) imageStore(w http.ResponseWriter, r *http.Request) {
	data := app.newTemplateData(r)
	data.Form = imageStoreForm{}

	app.render(w, r, http.StatusOK, "store.tmpl", data)
}

// ------------------------------------------------------------------------------
func (app *application) imageStorePost(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	year, err := strconv.Atoi(r.PostForm.Get("year"))
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	form := imageStoreForm{
		Location:    r.PostForm.Get("location"),
		Year:        year,
		People:      strings.Split(r.PostForm.Get("people"), "\r\n"),
		FieldErrors: map[string]string{},
	}

	if strings.TrimSpace(form.Location) == "" {
		form.FieldErrors["location"] = "this field cannot be blank"
	} else if utf8.RuneCountInString(form.Location) > 100 {
		form.FieldErrors["location"] = "this field cannot be more than 100 characters long"
	}

	if form.Year == 0 {
		form.FieldErrors["year"] = "this field cannot be 0"
	}

	if len(form.People) < 1 || len(form.People) > 5 {
		form.FieldErrors["people"] = "this field can only contain 1-5 lines"
	}

	if len(form.FieldErrors) > 0 {
		data := app.newTemplateData(r)
		data.Form = form
		app.render(w, r, http.StatusUnprocessableEntity, "store.tmpl", data)
		return
	}

	image := data.Image{
		Location: form.Location,
		Year:     form.Year,
		People:   form.People,
	}

	image, err = app.models.Images.Insert(image)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	app.sessionManager.Put(r.Context(), "flash", "imaged tucked!")

	http.Redirect(w, r, fmt.Sprintf("/images/%d", image.ID), http.StatusSeeOther)
}

// ------------------------------------------------------------------------------
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

// ------------------------------------------------------------------------------
func (app *application) userSignup(w http.ResponseWriter, r *http.Request) {
	data := app.newTemplateData(r)
	data.Form = userSignupForm{}

	app.render(w, r, http.StatusOK, "signup.tmpl", data)
}

// ------------------------------------------------------------------------------
func (app *application) userSignupPost(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "create a new user...")
}

// ------------------------------------------------------------------------------
func (app *application) userLogin(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "display a form for loggging in a user...")
}

// ------------------------------------------------------------------------------
func (app *application) userLoginPost(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "authenticate and login the user...")
}

// ------------------------------------------------------------------------------
func (app *application) userLogoutPost(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "logout the user...")
}
