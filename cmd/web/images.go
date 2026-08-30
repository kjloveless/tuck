package main

import (
	"errors"
	"fmt"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"tuck.loveless.dev/internal/data"
	"tuck.loveless.dev/internal/validator"
)

// add a storeImageHandler for the "POST /v1/images" endpoint. for now we
// simply return a plain-text placeholder response/
func (app *application) storeImageHandler(w http.ResponseWriter, r *http.Request) {
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil {
		app.badRequestResponse(w, r, errors.New("Content-Type must be application/json or multipart/form-data"))
		return
	}
	if mediaType == "multipart/form-data" {
		app.storeUploadedImageHandler(w, r)
		return
	}
	if mediaType != "application/json" {
		app.badRequestResponse(w, r, errors.New("Content-Type must be application/json or multipart/form-data"))
		return
	}

	app.storeImageJSONHandler(w, r)
}

func (app *application) storeImageJSONHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Location string   `json:"location"`
		Year     int      `json:"year"`
		People   []string `json:"people"`
	}

	err := app.readJSON(w, r, &input)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	image := data.Image{
		Location: input.Location,
		Year:     input.Year,
		People:   input.People,
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

func (app *application) storeUploadedImageHandler(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxImageBytes+(1<<20))
	if err := r.ParseMultipartForm(memThreshold); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}
	defer r.MultipartForm.RemoveAll()

	files := r.MultipartForm.File["images"]
	if len(files) != 1 {
		app.badRequestResponse(w, r, errors.New("provide exactly one image in the images field"))
		return
	}
	fileHeader := files[0]
	if fileHeader.Size > maxImageBytes {
		app.badRequestResponse(w, r, fmt.Errorf("image must not exceed %d bytes", maxImageBytes))
		return
	}
	extension, err := sniff(fileHeader)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}
	year, err := strconv.Atoi(r.FormValue("year"))
	if err != nil {
		app.badRequestResponse(w, r, errors.New("year must be an integer"))
		return
	}

	image := data.Image{
		Location: r.FormValue("location"),
		Year:     year,
		People:   r.MultipartForm.Value["people"],
	}
	v := validator.New()
	if data.ValidateImage(v, image); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	image.Path, err = app.storeBlob(candidate{fh: fileHeader, ext: extension})
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
	image, err = app.models.Images.Insert(image)
	if err != nil {
		cleanupErr := os.Remove(filepath.Join(app.dataDir(), image.Path))
		app.serverErrorResponse(w, r, errors.Join(err, cleanupErr))
		return
	}

	headers := make(http.Header)
	headers.Set("Location", fmt.Sprintf("/v1/images/%d", image.ID))
	if err := app.writeJSON(w, http.StatusCreated, envelope{"image": image}, headers); err != nil {
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
		Location *string  `json:"location"`
		Year     *int     `json:"year"`
		People   []string `json:"people"`
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
		Location string
		People   []string
		data.Filters
	}

	v := validator.New()

	qs := r.URL.Query()

	input.Location = app.readString(qs, "location", "")
	input.People = app.readCSV(qs, "people", []string{})

	input.Filters.Page = app.readInt(qs, "page", 1, v)
	input.Filters.PageSize = app.readInt(qs, "page_size", 20, v)

	input.Filters.Sort = app.readString(qs, "sort", "id")
	input.Filters.SortSafelist = []string{"id", "location", "year", "-id", "-location", "-year"}

	if data.ValidateFilters(v, input.Filters); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	images, metadata, err := app.models.Images.GetAll(input.Location, input.People, input.Filters)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	err = app.writeJSON(w, http.StatusOK, envelope{"images": images, "metadata": metadata}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}
