package main

import (
	"net/http"

	"github.com/julienschmidt/httprouter"
)

func (app *application) routes() http.Handler {
	router := httprouter.New()

	router.NotFound = http.HandlerFunc(app.notFoundResponse)
	router.MethodNotAllowed = http.HandlerFunc(app.methodNotAllowedResponse)

	router.HandlerFunc(http.MethodGet, "/v1/healthcheck", app.healthcheckHandler)

	router.HandlerFunc(http.MethodGet, "/v1/images", app.listImagesHandler)
	router.HandlerFunc(http.MethodPost, "/v1/images", app.storeImageHandler)
	router.HandlerFunc(http.MethodGet, "/v1/images/:id", app.showImageHandler)
	router.HandlerFunc(http.MethodPatch, "/v1/images/:id", app.updateImageHandler)
	router.HandlerFunc(http.MethodDelete, "/v1/images/:id", app.deleteImageHandler)

	router.HandlerFunc(http.MethodPost, "/v1/users", app.registerUserHandler)

	return app.recoverPanic(app.rateLimit(router))
}
