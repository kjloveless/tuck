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
		CurvePreferences: []tls.CurveID{tls.X25519, tls.CurveP256},
	}

	httpsServer := &http.Server{
		Addr:         fmt.Sprintf(":%d", app.config.port),
		Handler:      app.routes(),
		IdleTimeout:  time.Minute,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		ErrorLog:     slog.NewLogLogger(app.logger.Handler(), slog.LevelError),
		TLSConfig:    tlsConfig,
	}

	httpServer := &http.Server{
		Addr:         fmt.Sprintf(":%d", app.config.port-1),
		Handler:      app.redirectToHTTPS(),
		IdleTimeout:  time.Minute,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		ErrorLog:     slog.NewLogLogger(app.logger.Handler(), slog.LevelError),
	}

	shutdownError := make(chan error, 1)
	serverError := make(chan error, 2)

	go func() {
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		s := <-quit

		app.logger.Info("tucking in",
			"https_addr", httpsServer.Addr,
			"http_addr", httpServer.Addr,
			"signal", s.String(),
		)

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := httpServer.Shutdown(ctx); err != nil {
			shutdownError <- err
			return
		}

		if err := httpsServer.Shutdown(ctx); err != nil {
			shutdownError <- err
			return
		}

		app.logger.Info("completing background tasks..")

		app.wg.Wait()
		shutdownError <- nil
	}()

	go func() {
		app.logger.Info("HTTP redirector listening", "addr", httpServer.Addr)

		err := httpServer.ListenAndServe()
		if !errors.Is(err, http.ErrServerClosed) {
			serverError <- err
		}
	}()

	app.logger.Info("waking tuck", "addr", httpsServer.Addr, "env", app.config.env)

	err := httpsServer.ListenAndServeTLS("./tls/cert.pem", "./tls/key.pem")
	if !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	select {
	case err = <-serverError:
		return err

	case err = <-shutdownError:
		if err != nil {
			return err
		}
	}

	app.logger.Info("tucked in!")
	return nil
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
