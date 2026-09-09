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

	api := alice.New(app.authenticateToken)
	router.Handler(
		http.MethodGet, 
		"/v1/healthcheck", 
		api.ThenFunc(app.healthcheckHandler))

	router.Handler(
		http.MethodGet, 
		"/v1/images", 
		api.ThenFunc(app.requirePermission("images:read", app.listImagesHandler)))
	router.Handler(
		http.MethodPost, 
		"/v1/images", 
		api.ThenFunc(app.requirePermission("images:write", app.storeImageHandler)))
	router.Handler(
		http.MethodGet, 
		"/v1/images/:id", 
		api.ThenFunc(app.requirePermission("images:read", app.showImageHandler)))
	router.Handler(
		http.MethodPatch, 
		"/v1/images/:id", 
		api.ThenFunc(app.requirePermission("images:write", app.updateImageHandler)))
	router.Handler(
		http.MethodDelete, 
		"/v1/images/:id", 
		api.ThenFunc(app.requirePermission("images:write", app.deleteImageHandler)))

	router.Handler(
		http.MethodPost, 
		"/v1/users", 
		api.ThenFunc(app.registerUserHandler))
	router.Handler(
		http.MethodPut, 
		"/v1/users/activated", 
		api.ThenFunc(app.activateUserHandler))
	router.Handler(
		http.MethodPut, 
		"/v1/users/password", 
		api.ThenFunc(app.updateUserPasswordHandler))

	router.Handler(
		http.MethodPost, 
		"/v1/tokens/authentication", 
		api.ThenFunc(app.createAuthenticationTokenHandler))
	router.Handler(
		http.MethodPost, 
		"/v1/tokens/activation", 
		api.ThenFunc(app.createActivationTokenHandler))
	router.Handler(
		http.MethodPost, 
		"/v1/tokens/password-reset", 
		api.ThenFunc(app.createPasswordResetTokenHandler))

	dynamic := alice.New(
		app.sessionManager.LoadAndSave,
		app.authenticateSession,
	)
	protected := dynamic.Append(app.requireSession)

	router.Handler(
		http.MethodGet, 
		"/", 
		protected.ThenFunc(app.requirePermission("images:read", app.home)))
	router.Handler(
		http.MethodGet, 
		"/images/:id", 
		protected.ThenFunc(app.requirePermission("images:read", app.imageView)))
	router.Handler(
		http.MethodGet, 
		"/images/:id/file", 
		protected.ThenFunc(app.requirePermission("images:read", app.imageFile)))
	router.Handler(
		http.MethodGet, 
		"/image/store", 
		protected.ThenFunc(app.requirePermission("images:write", app.imageStore)))
	router.Handler(
		http.MethodPost, 
		"/image/store", 
		protected.ThenFunc(app.requirePermission("images:write", app.imageStorePost)))
	router.Handler(
		http.MethodPost, 
		"/images/:id/delete", 
		protected.ThenFunc(app.requirePermission("images:write", app.imageDelete)))

	router.Handler(http.MethodGet, "/user/signup", dynamic.ThenFunc(app.userSignup))
	router.Handler(http.MethodPost, "/user/signup", dynamic.ThenFunc(app.userSignupPost))
	router.Handler(http.MethodGet, "/user/login", dynamic.ThenFunc(app.userLogin))
	router.Handler(http.MethodPost, "/user/login", dynamic.ThenFunc(app.userLoginPost))
	router.Handler(http.MethodPost, "/user/logout", dynamic.ThenFunc(app.userLogoutPost))

	standard := alice.New(app.metrics, app.recoverPanic, app.enableCORS, app.rateLimit)

	router.Handler(http.MethodGet, "/debug/vars", expvar.Handler())

	return standard.Then(router)
}
