package main

import (
	"context"
	"net/http"

	"tuck.loveless.dev/internal/data"
)

type contextKey string

const authenticatedUserContextKey = contextKey("authenticatedUser")
const authenticatedDeviceContextKey = contextKey("authenticatedDevice")

func (app *application) contextSetAuthenticatedUser(r *http.Request, user *data.User) *http.Request {
	ctx := context.WithValue(r.Context(), authenticatedUserContextKey, user)
	return r.WithContext(ctx)
}

func (app *application) contextGetAuthenticatedUser(r *http.Request) (*data.User, bool) {
	user, ok := r.Context().Value(authenticatedUserContextKey).(*data.User)
	return user, ok
}

func (app *application) contextSetAuthenticatedDevice(r *http.Request, device *data.Device) *http.Request {
	ctx := context.WithValue(r.Context(), authenticatedDeviceContextKey, device)
	return r.WithContext(ctx)
}

func (app *application) contextGetAuthenticatedDevice(r *http.Request) (*data.Device, bool) {
	device, ok := r.Context().Value(authenticatedDeviceContextKey).(*data.Device)
	return device, ok
}
