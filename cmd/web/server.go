package main

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"
)

func (app *application) serve() error {
	tlsConfig := &tls.Config{
		MinVersion:       tls.VersionTLS12,
		CurvePreferences: []tls.CurveID{tls.X25519, tls.CurveP256},
		Certificates: []tls.Certificate{
			app.identity.TLSCertificate(),
		},
	}

	httpsServer := &http.Server{
		Addr:              fmt.Sprintf(":%d", app.config.port),
		Handler:           app.routes(),
		IdleTimeout:       time.Minute,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Minute,
		WriteTimeout:      10 * time.Minute,
		ErrorLog:          slog.NewLogLogger(app.logger.Handler(), slog.LevelError),
		TLSConfig:         tlsConfig,
	}
	httpServer := &http.Server{
		Addr:              fmt.Sprintf(":%d", app.config.port-1),
		Handler:           app.cleartextRoutes(),
		IdleTimeout:       time.Minute,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       5 * time.Second,
		WriteTimeout:      10 * time.Second,
		ErrorLog:          slog.NewLogLogger(app.logger.Handler(), slog.LevelError),
	}

	serverError := make(chan error, 2)

	go func() {
		app.logger.Info("HTTP redirector and local pairing page listening", "addr", httpServer.Addr)
		if err := httpServer.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			serverError <- fmt.Errorf("HTTP server: %w", err)
		}
	}()

	go func() {
		app.logger.Info(
			"waking tuck",
			"addr", httpsServer.Addr,
			"advertise_url", app.config.advertiseURL,
			"env", app.config.env,
			"pairing_url", fmt.Sprintf("http://localhost:%d/pair", app.config.port-1),
			"server_id", app.identity.ServerID(),
		)
		if err := httpsServer.ListenAndServeTLS("", ""); !errors.Is(err, http.ErrServerClosed) {
			serverError <- fmt.Errorf("HTTPS server: %w", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(quit)

	var serveErr error
	select {
	case serveErr = <-serverError:
		app.logger.Error("server stopped unexpectedly", "error", serveErr)
	case s := <-quit:
		app.logger.Info("tucking in", "signal", s.String())
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	shutdownErr := errors.Join(httpServer.Shutdown(ctx), httpsServer.Shutdown(ctx))

	app.logger.Info("completing background tasks")
	app.wg.Wait()

	if serveErr != nil {
		return errors.Join(serveErr, shutdownErr)
	}
	if shutdownErr != nil {
		return shutdownErr
	}
	app.logger.Info("tucked in!")
	return nil
}

func defaultAdvertiseURL(port int) string {
	addresses, err := net.InterfaceAddrs()
	if err == nil {
		for _, address := range addresses {
			ip, _, err := net.ParseCIDR(address.String())
			if err == nil && ip.To4() != nil && ip.IsPrivate() && !ip.IsLoopback() {
				return "https://" + net.JoinHostPort(ip.String(), strconv.Itoa(port))
			}
		}
	}
	return "https://" + net.JoinHostPort("localhost", strconv.Itoa(port))
}

// ------------------------------------------------------------------------------
func (app *application) redirectToHTTPS() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host := r.Host

		if hostname, _, err := net.SplitHostPort(host); err == nil {
			host = hostname
		}

		if app.config.port != 443 {
			host = net.JoinHostPort(host, strconv.Itoa(app.config.port))
		}

		target := "https://" + host + r.URL.RequestURI()

		app.logger.Info("redirecting a tucker!")
		http.Redirect(w, r, target, http.StatusPermanentRedirect)
	})
}
