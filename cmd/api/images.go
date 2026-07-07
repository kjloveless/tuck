package main

import (
	"fmt"
	"net/http"
	"time"

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

	fmt.Fprintf(w, "%+v\n", input)
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

	image := data.Image{
		ID:					id,
		CreatedAt:	time.Now(),
		Location:		"Mexico",
		Year:				2020,
		People:			[]string{"Kyle", "Ralitsa"},
		Version: 1,
	}

	// encode the struct to json and send it as the http response
	err = app.writeJSON(w, http.StatusOK, envelope{"image": image}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}
