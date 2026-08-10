package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"tuck.loveless.dev/internal/validator"

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

func (app *application) readJSON(w http.ResponseWriter, r *http.Request, dst any) error {
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

// readString() helper returns a string value from the query string, or the
// provided default value if no matching key could be found.
func (app *application) readString(qs url.Values, key string, defaultValue string) string {
	s := qs.Get(key)

	if s == "" {
		return defaultValue
	}

	return s
}

// readCSV() helper reads a string value from the query string and then splits
// it into a slice on the comma character. if no matching key could be found,
// it returns the provided default value
func (app *application) readCSV(qs url.Values, key string, defaultValue []string) []string {
	csv := qs.Get(key)

	if csv == "" {
		return defaultValue
	}

	return strings.Split(csv, ",")
}

// readInt() helper reads a string value from the query string and converts it
// to an integer before returning. if no matching key could be found it returns
// the provided default value. if the value couldn't be converted to an
// integer, then we record an error message in the provided Validator instance.
func (app *application) readInt(qs url.Values, key string, defaultValue int, v *validator.Validator) int {
	s := qs.Get(key)

	if s == "" {
		return defaultValue
	}

	i, err := strconv.Atoi(s)
	if err != nil {
		v.AddError(key, "must be an integer value")
		return defaultValue
	}

	return i
}

func (app *application) background(fn func()) {
	app.wg.Go(func() {
		defer func() {
			pv := recover()
			if pv != nil {
				app.logger.Error(fmt.Sprintf("%v", pv))
			}
		}()

		fn()
	})
}

// /-----------------------------------------------------------------------------
func (app *application) render(
	w http.ResponseWriter,
	r *http.Request,
	status int,
	page string,
	data templateData,
) {
	ts, ok := app.templateCache[page]
	if !ok {
		err := fmt.Errorf("the template %s does not exist", page)
		app.serverErrorResponse(w, r, err)
		return
	}

	buf := new(bytes.Buffer)

	err := ts.ExecuteTemplate(buf, "base", data)
	if err != nil {
		err = fmt.Errorf("zoinks! %s", err)
		app.serverErrorResponse(w, r, err)
		return
	}

	w.WriteHeader(status)

	buf.WriteTo(w)
}

// /-----------------------------------------------------------------------------
func (app *application) newTemplateData(r *http.Request) templateData {
	return templateData{
		CurrentYear: time.Now().Year(),
		Flash:       app.sessionManager.PopString(r.Context(), "flash"),
	}
}

//------------------------------------------------------------------------------
func sniff(fh *multipart.FileHeader) (string, error) {
	f, err := fh.Open()
	if err != nil {
		return "", err
	}
	defer f.Close()

	head := make([]byte, 512)
	n, err := io.ReadFull(f, head)
	if err != nil && !errors.Is(err, io.EOF) && !errors.Is(err, io.ErrUnexpectedEOF) {
		return "", err
	}

	ext, ok := allowedImageTypes[http.DetectContentType(head[:n])]
	if !ok {
		return "", errors.New("unsupported content type")
	}
	return ext, nil
}

//------------------------------------------------------------------------------
func (app application) tmpDir() string {
	return "./tmp/"
}

func (app application) dataDir() string {
	return "./data/images/"
}

//------------------------------------------------------------------------------
func (app *application) storeBlob(c candidate) (string, error) {
	f, err := c.fh.Open()
	if err != nil {
		return "", err
	}
	defer f.Close()

	tmp, err := os.CreateTemp(app.tmpDir(), "up-*")
	if err != nil {
		return "", err
	}

	committed := false
	defer func() {
		tmp.Close()
		if !committed {
			os.Remove(tmp.Name())
		}
	}()

	h := sha256.New()
	if _, err := io.Copy(io.MultiWriter(tmp, h), f); err != nil {
		return "", err
	}
	if err := tmp.Close(); err != nil {
		return "", err
	}

	sum := hex.EncodeToString(h.Sum(nil))
	rel := filepath.Join(sum[:2], sum[2:4], sum+c.ext)
	dst := filepath.Join(app.dataDir(), rel)

	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return "", err
	}
	if err := os.Rename(tmp.Name(), dst); err != nil {
		return "", err
	}

	committed = true
	return rel, nil
}

//------------------------------------------------------------------------------
func splitLines(s string) []string {
	var out []string
	for _, line := range strings.Split(strings.ReplaceAll(s, "\r\n", "\n"), "\n") {
		if line = strings.TrimSpace(line); line != "" {
			out = append(out, line)
		}
	}
	return out
}
