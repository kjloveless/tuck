package main

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/julienschmidt/httprouter"
)

// add a createImageHandler for the "POST /v1/images" endpoint. for now we
// simply return a plain-text placeholder response/
func (app *application) storeImageHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "store a new image")
}

// add a showImageHandler for the "GET /v1/images/:id" endpoint. for now, we
// retrieve the interpolated "id" parameter from the curretn url and include it
// in a placeholder response
func (app *application) showImageHandler(w http.ResponseWriter, r *http.Request) {
	// when httprouter is parssing a request, any interpolated url parameters
	// will be stored in the request context. we can use the ParamsFromContext()
	// function to retrieve a slice containing these paramter names and values
	params := httprouter.ParamsFromContext(r.Context())

	// we can then use the ByName() method to get the value of the "id" parameter
	// from the slice. in our project all images will have a unique positive
	// integer ID, but the value returned by ByName() is always a string. so we
	// try to convert it to an integer. if the parameter couldn't be converted,
	// or is less than 1, we know the id is invalid so we use the http.NotFound()
	// function to return a 404 Not Found response.
	id, err := strconv.Atoi(params.ByName("id"))
	if err != nil || id < 1 {
		http.NotFound(w, r)
		return
	}

	// otherwise, interpolate the image if in a placeholder response
	fmt.Fprintf(w, "show the details of image %d\n", id)
}
