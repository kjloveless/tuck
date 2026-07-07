package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/julienschmidt/httprouter"
)

type envelope map[string]any

// retrieve the "id" url parameter from the current request context, then
// convert it to an integer and return it. if the operation isn't successful,
// return 0 and an error.
func (app *application) readIDParam(r *http.Request) (int, error) {
	params := httprouter.ParamsFromContext(r.Context())

	id, err := strconv.Atoi(params.ByName("id"))
	if err != nil || id < 1 {
		return 0, errors.New("invalid id parameter")
	}

	return id, nil
}

func(app *application) readJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(w, r.Body, 1_048_576)

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	err := dec.Decode(dst)
	if err != nil {
		var syntaxError *json.SyntaxError
		var unmarshalTypeError *json.UnmarshalTypeError
		var invalidUnmarshalError *json.InvalidUnmarshalError
		var maxBytesError *http.MaxBytesError

		switch {
		case errors.As(err, &syntaxError):
			return fmt.Errorf("body contains badly-formed json (at character %d)", syntaxError.Offset)

		case errors.Is(err, io.ErrUnexpectedEOF):
			return errors.New("body contains badly-formed json")

		case errors.As(err, &unmarshalTypeError):
			if unmarshalTypeError.Field != "" {
				return fmt.Errorf("body contains incorrect json type for field %q", unmarshalTypeError.Field)
			}
			return fmt.Errorf("body contains incorrect json type (at character %d)", unmarshalTypeError.Offset)

		case errors.Is(err, io.EOF):
			return errors.New("body must not be empty")

		case strings.HasPrefix(err.Error(), "json: unknown field "):
			fieldName := strings.TrimPrefix(err.Error(), "json: unknown field ")
			return fmt.Errorf("body contains unknown key %s", fieldName)

		case errors.As(err, &maxBytesError):
			return fmt.Errorf("body must not be larger than %d bytes", maxBytesError.Limit)

		case errors.As(err, &invalidUnmarshalError):
			panic(err)

		default:
			return err
		}
	}

	err = dec.Decode(&struct{}{})
	if !errors.Is(err, io.EOF) {
		return errors.New("body must only contain a single json value")
	}

	return nil
}

// define a writejson() helper for sending responses. this takes the
// destination http.ResposneWriter, the http status code to send, the data to
// encode to json, and a headers map containing any additional http headers we
// want to include in the response
func (app *application) writeJSON(w http.ResponseWriter, status int, data envelope, headers http.Header) error {
	// encode the data to json, returning the error if there was one
	js, err := json.Marshal(data)
	if err != nil {
		return err
	}

	// append a newline to make it easier to view in terminal applications
	js = append(js, '\n')

	// at this point, we know that we won't encounter any more errors before
	// writing the resposne, so it's safe to add any headers that we atnt to
	// include. we loop through the headers map (which behind the scenes has the
	// type map[string][]string) and add all the header keys and values to the
	// http.ResponseWriter's header map. note that it's ok if the provided
	// headers map is nil. go doesn't throw an error if you try to range over (or
	// generally, read fromo) a nil map
	for key, values := range headers {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}

	// add the "Content-Type: application/json" header, then write the status
	// code and json resposne
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write(js)

	return nil
}
