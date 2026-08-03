package main

import (
	"expvar"
	"net/http"

	"github.com/julienschmidt/httprouter"
)

func (app *application) routes() http.Handler {
	router := httprouter.New()

	router.NotFound = http.HandlerFunc(app.notFoundResponse)
	router.MethodNotAllowed = http.HandlerFunc(app.methodNotAllowedResponse)

	router.HandlerFunc(http.MethodGet, "/v1/healthcheck", app.healthcheckHandler)

	router.HandlerFunc(http.MethodGet, "/v1/images", app.requirePermission("images:read", app.listImagesHandler))
	router.HandlerFunc(http.MethodPost, "/v1/images", app.requirePermission("images:write", app.storeImageHandler))
	router.HandlerFunc(http.MethodGet, "/v1/images/:id", app.requirePermission("images:read", app.showImageHandler))
	router.HandlerFunc(http.MethodPatch, "/v1/images/:id", app.requirePermission("images:write", app.updateImageHandler))
	router.HandlerFunc(http.MethodDelete, "/v1/images/:id", app.requirePermission("images:write", app.deleteImageHandler))

	router.HandlerFunc(http.MethodPost, "/v1/users", app.registerUserHandler)
	router.HandlerFunc(http.MethodPut, "/v1/users/activated", app.activateUserHandler)
	router.HandlerFunc(http.MethodPut, "/v1/users/password", app.updateUserPasswordHandler)

	router.HandlerFunc(http.MethodPost, "/v1/tokens/authentication", app.createAuthenticationTokenHandler)
	router.HandlerFunc(http.MethodPost, "/v1/tokens/activation", app.createActivationTokenHandler)
	router.HandlerFunc(http.MethodPost, "/v1/tokens/password-reset", app.createPasswordResetTokenHandler)

	router.HandlerFunc(http.MethodGet, "/", app.home)
	router.HandlerFunc(http.MethodGet, "/images/:id", app.imageView)
	router.HandlerFunc(http.MethodGet, "/images", app.imageStore)


	router.Handler(http.MethodGet, "/debug/vars", expvar.Handler())

	return app.metrics(app.recoverPanic(app.enableCORS(app.rateLimit(app.authenticate(router)))))
}
