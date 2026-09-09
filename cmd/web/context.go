package main

import (
	"context"
	"net/http"

	"tuck.loveless.dev/internal/data"
)

type contextKey string

const authenticatedUserContextKey = contextKey("authenticatedUser")
const userPermissionsContextKey = contextKey("userPermissions")

func (app *application) contextSetUserPermissions(r *http.Request, permissions data.Permissions) *http.Request {
	ctx := context.WithValue(r.Context(), userPermissionsContextKey, permissions)
	return r.WithContext(ctx)
}

func (app *application) contextGetUserPermissions(r *http.Request) data.Permissions {
	permissions, _ := r.Context().Value(userPermissionsContextKey).(data.Permissions)
	return permissions
}

func (app *application) contextSetAuthenticatedUser(r *http.Request, user *data.User) *http.Request {
	ctx := context.WithValue(r.Context(), authenticatedUserContextKey, user)
	return r.WithContext(ctx)
}

func (app *application) contextGetAuthenticatedUser(r *http.Request) (*data.User, bool) {
	user, ok := r.Context().Value(authenticatedUserContextKey).(*data.User)
	return user, ok
}
