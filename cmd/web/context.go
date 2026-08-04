package main

import (
	"context"
	"net/http"

	"tuck.loveless.dev/internal/data"
)

type contextKey string

const authenticatedUserContextKey = contextKey("authenticatedUser")

func (app *application) contextSetAuthenticatedUser(r *http.Request, user *data.User) *http.Request {
	ctx := context.WithValue(r.Context(), authenticatedUserContextKey, user)
	return r.WithContext(ctx)
}

func (app *application) contextGetAuthenticatedUser(r *http.Request) (*data.User, bool) {
	user, ok := r.Context().Value(authenticatedUserContextKey).(*data.User)
	return user, ok
}
