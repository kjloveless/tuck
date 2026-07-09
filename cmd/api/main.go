package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"tuck.loveless.dev/internal/data"

	_ "modernc.org/sqlite"
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
	db 		struct {
		dsn 					string
		maxOpenConns	int
		maxIdleConns	int
		maxIdleTime		time.Duration
	}
}

// define an application struct to hold the dependencies for our http handlers,
// helpers, and middleware. at the moment this only contains a copy of the
// config struct and a logger, but it will grow to include a lot more as our
// build progresses.
type application struct {
	config	config
	logger	*slog.Logger
	models	data.Models
}

func main() {
	// decalre an instance of the config struct
	var cfg config

	// read the value of the port and env command-line falgs into the config
	// struct. we default to using the port number 5375 and the environment
	// "development" if no corresponding flags are provided
	flag.IntVar(&cfg.port, "port", 5375, "api server port")
	flag.StringVar(&cfg.env, "env", "development", "environment (development|staging|production)")

	flag.StringVar(&cfg.db.dsn, "db-dsn", os.Getenv("TUCK_DB_DSN"), "sqlite data source name")
	flag.IntVar(&cfg.db.maxOpenConns, "db-max-open-conns", 1, "sqlite max open connections")
	flag.IntVar(&cfg.db.maxIdleConns, "db-max-idle-conns", 1, "sqlite max idle connections")
	flag.DurationVar(&cfg.db.maxIdleTime, "db-max-idle-time", 15*time.Minute, "sqlite max connection idle time")

	flag.Parse()

	// initialize a new structured logger which writes log entries to the
	// standard out stream
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	db, err := openDB(cfg)
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}
	defer db.Close()

	logger.Info("database connection pool established")

	// declare an instance of the application struct, containing the config
	// struct, logger, and models
	app := &application{
		config: cfg,
		logger: logger,
		models: data.NewModels(db),
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

	err = srv.ListenAndServe()
	logger.Error(err.Error())
	os.Exit(1)
}

func openDB(cfg config) (*sql.DB, error) {
	// use sql.Open() to create an empty connection pool, using the dsn from the
	// config struct
	db, err := sql.Open("sqlite", cfg.db.dsn)
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(cfg.db.maxOpenConns)
	db.SetMaxIdleConns(cfg.db.maxIdleConns)
	db.SetConnMaxIdleTime(cfg.db.maxIdleTime)

	// create a context with a 5-second timeout deadline
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// use pingcontext to establish a new connection to the database, passing in
	// the context we created above as a parameter. if the connection couldn't be
	// established successsfully within the 5-second deadline, then this will
	// return an error. if we get this error, or any other, we close the
	// connection pool and return the error
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, err
	}


	//wal mode allows reads to continue while a write is taking place. the
	//setting is persistent for this database file, so it does not need to be
	//enabled separately on every connection
	var journalMode string

	if err = db.QueryRowContext(ctx, `PRAGMA journal_mode = WAL`).Scan(&journalMode); err != nil {
		db.Close()
		return nil, fmt.Errorf("enabling WAL mode: %w", err)
	}

	if journalMode != "wal" {
		db.Close()
		return nil, fmt.Errorf("enabling WAL mode: sqlite returned journal mode %q", journalMode)
	}

	// return the sql.DB connection pool
	return db, nil
}
