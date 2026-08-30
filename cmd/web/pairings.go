package main

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"html/template"
	"net"
	"net/http"
	"time"

	"tuck.loveless.dev/internal/data"

	qrcode "github.com/skip2/go-qrcode"
)

type pairingPayload struct {
	Version             int       `json:"v"`
	ServerID            string    `json:"server_id"`
	Endpoint            string    `json:"endpoint"`
	CACertificateSHA256 string    `json:"ca_cert_sha256"`
	Secret              string    `json:"secret"`
	ExpiresAt           time.Time `json:"expires_at"`
}

var pairingPageTemplate = template.Must(template.New("pairing").Parse(`<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width,initial-scale=1">
  <title>Pair Tuck</title>
  <style>
    :root { color-scheme: light dark; font-family: system-ui, sans-serif; }
    body { display: grid; min-height: 100vh; margin: 0; place-items: center; text-align: center; }
    main { max-width: 32rem; padding: 2rem; }
    img { background: white; border-radius: 1rem; max-width: min(80vw, 384px); padding: .75rem; width: 100%; }
    code { overflow-wrap: anywhere; }
  </style>
</head>
<body>
  <main>
    <h1>Pair with Tuck</h1>
    <p>Scan this code in the Tuck app. It expires at {{.ExpiresAt}}.</p>
    <img alt="Tuck pairing QR code" src="data:image/png;base64,{{.QRCode}}">
    <p><code>{{.Endpoint}}</code></p>
  </main>
</body>
</html>`))

func (app *application) pairingPage(w http.ResponseWriter, r *http.Request) {
	if !requestIsLoopback(r) {
		app.notFoundResponse(w, r)
		return
	}
	if app.config.advertiseURL == "" {
		http.Error(w, "start Tuck with -advertise-url before pairing", http.StatusServiceUnavailable)
		return
	}

	if err := app.models.Pairings.DeleteExpired(); err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
	session, err := app.models.Pairings.New(5 * time.Minute)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
	payload := pairingPayload{
		Version:             1,
		ServerID:            app.identity.ServerID(),
		Endpoint:            app.config.advertiseURL,
		CACertificateSHA256: app.identity.CACertificateSHA256(),
		Secret:              session.Plaintext,
		ExpiresAt:           session.ExpiresAt,
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
	png, err := qrcode.Encode(string(payloadJSON), qrcode.Medium, 384)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Security-Policy", "default-src 'none'; img-src data:; style-src 'unsafe-inline'")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	err = pairingPageTemplate.Execute(w, map[string]string{
		"Endpoint":  payload.Endpoint,
		"ExpiresAt": payload.ExpiresAt.Local().Format(time.Kitchen),
		"QRCode":    base64.StdEncoding.EncodeToString(png),
	})
	if err != nil {
		app.logError(r, err)
	}
}

func (app *application) createPairingHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Secret     string `json:"secret"`
		DeviceName string `json:"device_name"`
	}
	if err := app.readJSON(w, r, &input); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	device, authenticationToken, err := app.models.Pairings.Consume(input.Secret, input.DeviceName)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrInvalidPairingToken):
			app.errorResponse(w, r, http.StatusUnauthorized, "invalid or expired pairing code")
		case errors.Is(err, data.ErrInvalidDeviceName):
			app.badRequestResponse(w, r, err)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	if err := app.writeJSON(w, http.StatusCreated, envelope{
		"device":               device,
		"authentication_token": authenticationToken,
	}, nil); err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *application) cleartextRoutes() http.Handler {
	redirect := app.redirectToHTTPS()
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/pair" && requestIsLoopback(r) {
			app.pairingPage(w, r)
			return
		}
		redirect.ServeHTTP(w, r)
	})
}

func requestIsLoopback(r *http.Request) bool {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return false
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}
