package main

import (
	"net/http"

	"github.com/julienschmidt/httprouter"
)

func (app *application) routes() http.Handler {
	// initialize a new httprouter router instance
	router := httprouter.New()

	// register the relevant methods, url patterns and handler functions for our
	// endpoints using the HandlerFunc() method. note that the http.MethodGet and
	// http.MethodPost are constants which equate to the strings "GET" and "POST"
	// respectively
	router.HandlerFunc(http.MethodGet, "/v1/healthcheck", app.healthcheckHandler)
	router.HandlerFunc(http.MethodPost, "/v1/images", app.storeImageHandler)
	router.HandlerFunc(http.MethodGet, "/v1/images/:id", app.showImageHandler)

	// return the httprouter instance
	return router
}
