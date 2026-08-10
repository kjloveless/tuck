package main

import (
	"errors"
	"fmt"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"tuck.loveless.dev/internal/data"
)

const (
	maxUploadBytes 	= 256 << 20
	maxImageBytes 	= 32 << 20
	maxImages				= 20
	memThreshold		= 10 << 20
)

var allowedImageTypes = map[string]string{
	"image/jpeg": ".jpg",
	"image/png":	".png",
	"image/webp":	".webp",
	"image/gif": 	".gif",
}

type imageStoreForm struct {
	Location    string
	YearRaw			string
	Year        int
	PeopleRaw		string
	People      []string
	FieldErrors map[string]string
}

type candidate struct {
	fh 	*multipart.FileHeader
	ext	string
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
	r.Body = http.MaxBytesReader(w, r.Body, maxUploadBytes)

	form := imageStoreForm{ FieldErrors: map[string]string{} }
	
	if err := r.ParseMultipartForm(memThreshold); err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			data := app.newTemplateData(r)
			data.Form = form
			form.FieldErrors["images"] = "those images are too large in total"
			app.render(w, r, http.StatusUnprocessableEntity, "store.tmpl", data)
			return
		}
		app.badRequestResponse(w, r, err)
		return
	}

	form.Location 	= r.PostForm.Get("location")
	form.YearRaw 		= strings.TrimSpace(r.PostForm.Get("year"))
	form.PeopleRaw	= r.PostForm.Get("people")
	form.People			= splitLines(form.PeopleRaw)

	if strings.TrimSpace(form.Location) == "" {
		form.FieldErrors["location"] = "this field cannot be blank"
	} else if utf8.RuneCountInString(form.Location) > 100 {
		form.FieldErrors["location"] = "this field cannot be more than 100 characters long"
	}

	year, err := strconv.Atoi(form.YearRaw)
	thisYear := time.Now().Year()

	switch {
	case form.YearRaw == "":
		form.FieldErrors["year"] = "this field cannot be blank"
	case err != nil:
		form.FieldErrors["year"] = err.Error()
	case year < 1826 || thisYear < year:
		form.FieldErrors["year"] = fmt.Sprintf("this field must be between 1826 and %d", thisYear)
	default:
		form.Year = year
	}

	if len(form.People) < 1 || len(form.People) > 5 {
		form.FieldErrors["people"] = "this field can only contain 1-5 lines"
	}

	files := r.MultipartForm.File["images"]
	candidates := make([]candidate, 0, len(files))

	switch {
	case len(files) == 0:
		form.FieldErrors["images"] = "choose at least one image"
	case len(files) > maxImages:
		form.FieldErrors["images"] = fmt.Sprintf("no more than %d images at a time", maxImages)
	default:
		for _, fh := range files {
			if fh.Size > maxImageBytes {
				form.FieldErrors["images"] = fmt.Sprintf("%q is too large", fh.Filename)
				break
			}
			ext, err := sniff(fh)
			if err != nil {
				form.FieldErrors["images"] = fmt.Sprintf("%q is not a supported image", fh.Filename)
				break
			}
			candidates = append(candidates, candidate{ fh: fh, ext: ext })
		}
	}

	if len(form.FieldErrors) > 0 {
		data := app.newTemplateData(r)
		data.Form = form
		app.render(w, r, http.StatusUnprocessableEntity, "store.tmpl", data)
		return
	}

	paths := make([]string, 0, len(candidates))
	for _, c := range candidates {
		p, err := app.storeBlob(c)
		if err != nil {
			app.serverErrorResponse(w, r, err)
			return
		}
		paths = append(paths, p)
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
	}

	app.sessionManager.Put(r.Context(), "flash", "imaged tucked!")

	http.Redirect(w, r, "/", http.StatusSeeOther)
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
