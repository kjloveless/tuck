package main

import (
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"
)

// declare a string containing the application version number. later we'll
// generate this automatically at build time, but for now we'll just store the
// version number as a hard-coded global constant
const version = "0.0.1"

// define a config struct to hold all the configuration settings for our
// application. for now, the only configuration settings will be the network
// port that we want the server to listen on, and the name of the current
// operating envirnoment for the application (development, staging, production,
// etc...). we ill read in these configuration settings from command-line flags
// when the application starts.
type config struct {
	port	int
	env 	string
}

// define an application struct to hold the dependencies for our http handlers,
// helpers, and middleware. at the moment this only contains a copy of the
// config struct and a logger, but it will grow to include a lot more as our
// build progresses.
type application struct {
	config	config
	logger	*slog.Logger
}

func main() {
	// decalre an instance of the config struct
	var cfg config

	// read the value of the port and env command-line falgs into the config
	// struct. we default to using the port number 5375 and the environment
	// "development" if no corresponding flags are provided
	flag.IntVar(&cfg.port, "port", 5375, "api server port")
	flag.StringVar(&cfg.env, "env", "development", "environment (development|staging|production)")
	flag.Parse()

	// initialize a new structured logger which writes log entries to the
	// standard out stream
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	// declare an instance of the application struct, containing the config
	// struct and the logger
	app := &application{
		config: cfg,
		logger: logger,
	}

	// declare a http server which listens on the port provided in the config
	// struct, uses the httprouter instance returned by app.routes() as the
	// server handler, has some sensible timeout settings, and writes any log 
	// messages to the structured logger at Error level.
	srv := &http.Server{
		Addr:						fmt.Sprintf(":%d", cfg.port),
		Handler:				app.routes(),
		IdleTimeout:		time.Minute,
		ReadTimeout:		5 * time.Second,
		WriteTimeout: 	10 * time.Second,
		ErrorLog:				slog.NewLogLogger(logger.Handler(), slog.LevelError),
	}

	// start the http server
	logger.Info("waking tuck", "addr", srv.Addr, "env", cfg.env)

	err := srv.ListenAndServe()
	logger.Error(err.Error())
	os.Exit(1)
}
