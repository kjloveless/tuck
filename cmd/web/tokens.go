package main

import (
	"errors"
	"net/http"
	"time"

	"tuck.loveless.dev/internal/data"
	"tuck.loveless.dev/internal/validator"
)

func (app *application) createAuthenticationTokenHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	err := app.readJSON(w, r, &input)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	v := validator.New()

	data.ValidateUsername(v, input.Username)
	data.ValidatePasswordPlaintext(v, input.Password)

	if !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	user, err := app.models.Users.Authenticate(input.Username, input.Password)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrInvalidCredentials):
			app.invalidCredentialsResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	token, err := app.models.Tokens.New(user.ID, 24*time.Hour, data.ScopeAuthentication)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	err = app.writeJSON(w, http.StatusCreated, envelope{"authentication_token": token}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *application) createPasswordResetTokenHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Username string `json:"username"`
	}

	err := app.readJSON(w, r, &input)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	v := validator.New()

	if data.ValidateUsername(v, input.Username); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	user, err := app.models.Users.GetByUsername(input.Username)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			v.AddError("username", "no matching username found")
			app.failedValidationResponse(w, r, v.Errors)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	if !user.Activated {
		v.AddError("username", "user account must be activated")
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	_, err = app.models.Tokens.New(user.ID, 45*time.Minute, data.ScopePasswordReset)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	err = app.writeJSON(w, http.StatusAccepted, envelope{}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *application) createActivationTokenHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Username string `json:"username"`
	}

	err := app.readJSON(w, r, &input)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	v := validator.New()

	if data.ValidateUsername(v, input.Username); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	user, err := app.models.Users.GetByUsername(input.Username)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			v.AddError("username", "no matching username found")
			app.failedValidationResponse(w, r, v.Errors)
		default:
			app.serverErrorResponse(w, r, err)
		}

		return
	}

	if user.Activated {
		v.AddError("username", "user has already been activated")
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	_, err = app.models.Tokens.New(user.ID, 3*24*time.Hour, data.ScopeActivation)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	err = app.writeJSON(w, http.StatusAccepted, envelope{}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}
