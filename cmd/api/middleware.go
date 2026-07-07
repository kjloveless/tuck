package main

import (
	"fmt"
	"net/http"
)

func (app *application) recoverPanic(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// create a deferred function (which will always be run in the event of a
		// panic).
		defer func() {
			// use the built-in recover() function to check if a panic occurred.
			// if a panic did happen, recover() will return the panic value. if a
			// panic didn't happen, it will return nil.
			pv := recover()
			if pv != nil {
				// if there was a panic, set a "Connection: close" header on the
				// response. this acts as a trigger to make go's http server
				// automatically close the current connection after the response has
				// been sent
				w.Header().Set("Connection", "close")
				// the value returned by recover() has the type any, so we use
				// fmt.Errorf() with the %v verb to coerce it into an error and call
				// our serverErrorResponse() helper. in turn, this will log the error
				// at the ERROR level and send the client a 500 internal server error
				// response
				app.serverErrorResponse(w, r, fmt.Errorf("%v", pv))
			}
		}()

		next.ServeHTTP(w, r)
	})
}
