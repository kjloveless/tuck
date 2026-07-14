package main

import (
	"errors"
	"fmt"
	"net/http"

	"tuck.loveless.dev/internal/data"
	"tuck.loveless.dev/internal/validator"
)

// add a storeImageHandler for the "POST /v1/images" endpoint. for now we
// simply return a plain-text placeholder response/
func (app *application) storeImageHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Location		string		`json:"location"`
		Year				int				`json:"year"`
		People			[]string	`json:"people"`
	}

	err := app.readJSON(w, r, &input) 
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	image := data.Image{
		Location: input.Location,
		Year: 		input.Year,
		People: 	input.People,
	}

	v := validator.New()

	// call the validateimage() function and if any checks fail, return a
	// response
	if data.ValidateImage(v, image); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	image, err = app.models.Images.Insert(image)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	headers := make(http.Header)
	headers.Set("Location", fmt.Sprintf("/v1/images/%d", image.ID))

	err = app.writeJSON(w, http.StatusCreated, envelope{"image": image}, headers)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

// add a showImageHandler for the "GET /v1/images/:id" endpoint. for now, we
// retrieve the interpolated "id" parameter from the curretn url and include it
// in a placeholder response
func (app *application) showImageHandler(w http.ResponseWriter, r *http.Request) {
	id, err := app.readIDParam(r)	
	if err != nil {
		app.notFoundResponse(w, r)
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

	// encode the struct to json and send it as the http response
	err = app.writeJSON(w, http.StatusOK, envelope{"image": image}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *application) updateImageHandler(w http.ResponseWriter, r *http.Request) {
	id, err := app.readIDParam(r)
	if err != nil {
		app.notFoundResponse(w, r)
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

	var input struct {
		Location 	*string 		`json:"location"`
		Year			*int				`json:"year"`
		People		[]string	`json:"people"`
	}

	err = app.readJSON(w, r, &input)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	if input.Location != nil {
		image.Location = *input.Location
	}

	if input.Year != nil {
		image.Year = *input.Year
	}

	if input.People != nil {
		image.People = input.People
	}

	v := validator.New()

	if data.ValidateImage(v, image); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	image, err = app.models.Images.Update(image)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrEditConflict):
			app.editConflictResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	err = app.writeJSON(w, http.StatusOK, envelope{"image": image}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *application) deleteImageHandler(w http.ResponseWriter, r *http.Request) {
	id, err := app.readIDParam(r)
	if err != nil {
		app.notFoundResponse(w, r)
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

	err = app.writeJSON(w, http.StatusOK, envelope{"message": "image successfully deleted"}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *application) listImagesHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Location	string
		People		[]string
		data.Filters
	}

	v := validator.New()

	qs := r.URL.Query()

	input.Location = app.readString(qs, "location", "")
	input.People = app.readCSV(qs, "people", []string{})

	input.Filters.Page = app.readInt(qs, "page", 1, v)
	input.Filters.PageSize = app.readInt(qs, "page_size", 20, v)

	input.Filters.Sort = app.readString(qs, "sort", "id")

	if !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	fmt.Fprintf(w, "%+v\n", input)
}
