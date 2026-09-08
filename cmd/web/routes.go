package main

import (
	"expvar"
	"net/http"

	"github.com/julienschmidt/httprouter"
	"github.com/justinas/alice"
)

func (app *application) routes() http.Handler {
	router := httprouter.New()
	fileServer := http.FileServer(http.Dir("./ui/static"))

	router.NotFound = http.HandlerFunc(app.notFoundResponse)
	router.MethodNotAllowed = http.HandlerFunc(app.methodNotAllowedResponse)

	router.Handler(http.MethodGet, "/static/*filepath", http.StripPrefix("/static", fileServer))

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

	dynamic := alice.New(app.sessionManager.LoadAndSave)

	router.Handler(http.MethodGet, "/", dynamic.ThenFunc(app.home))
	router.Handler(http.MethodGet, "/images/:id", dynamic.ThenFunc(app.imageView))
	router.Handler(http.MethodGet, "/images/:id/file", dynamic.ThenFunc(app.imageFile))
	router.Handler(http.MethodGet, "/image/store", dynamic.ThenFunc(app.imageStore))
	router.Handler(http.MethodPost, "/image/store", dynamic.ThenFunc(app.imageStorePost))
	router.Handler(http.MethodPost, "/images/:id/delete", dynamic.ThenFunc(app.imageDelete))

	router.Handler(http.MethodGet, "/user/signup", dynamic.ThenFunc(app.userSignup))
	router.Handler(http.MethodPost, "/user/signup", dynamic.ThenFunc(app.userSignupPost))
	router.Handler(http.MethodGet, "/user/login", dynamic.ThenFunc(app.userLogin))
	router.Handler(http.MethodPost, "/user/login", dynamic.ThenFunc(app.userLoginPost))
	router.Handler(http.MethodPost, "/user/logout", dynamic.ThenFunc(app.userLogoutPost))

	standard := alice.New(app.metrics, app.recoverPanic, app.enableCORS, app.rateLimit, app.authenticate)

	router.Handler(http.MethodGet, "/debug/vars", expvar.Handler())

	return standard.Then(router)
}
