package main

import (
	"context"
	"database/sql"
	"expvar"
	"flag"
	"fmt"
	"html/template"
	"log/slog"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"

	"tuck.loveless.dev/internal/data"
	"tuck.loveless.dev/internal/mailer"
	"tuck.loveless.dev/internal/vcs"

	_ "modernc.org/sqlite"
)

var (
	version = vcs.Version()
)

// define a config struct to hold all the configuration settings for our
// application. for now, the only configuration settings will be the network
// port that we want the server to listen on, and the name of the current
// operating envirnoment for the application (development, staging, production,
// etc...). we ill read in these configuration settings from command-line flags
// when the application starts.
type config struct {
	port int
	env  string
	db   struct {
		dsn          string
		maxOpenConns int
		maxIdleConns int
		maxIdleTime  time.Duration
	}
	limiter struct {
		rps     float64
		burst   int
		enabled bool
	}
	smtp struct {
		host     string
		port     int
		username string
		password string
		sender   string
	}
	cors struct {
		trustedOrigins []string
	}
}

// define an application struct to hold the dependencies for our http handlers,
// helpers, and middleware. at the moment this only contains a copy of the
// config struct and a logger, but it will grow to include a lot more as our
// build progresses.
type application struct {
	config        config
	logger        *slog.Logger
	models        data.Models
	mailer        *mailer.Mailer
	templateCache map[string]*template.Template
	wg            sync.WaitGroup
}

func main() {
	var cfg config

	flag.IntVar(&cfg.port, "port", 5375, "api server port")
	flag.StringVar(&cfg.env, "env", "development", "environment (development|staging|production)")

	flag.StringVar(&cfg.db.dsn, "db-dsn", "", "sqlite data source name")
	flag.IntVar(&cfg.db.maxOpenConns, "db-max-open-conns", 50, "sqlite max open connections")
	flag.IntVar(&cfg.db.maxIdleConns, "db-max-idle-conns", 50, "sqlite max idle connections")
	flag.DurationVar(&cfg.db.maxIdleTime, "db-max-idle-time", 15*time.Minute, "sqlite max connection idle time")

	flag.Float64Var(&cfg.limiter.rps, "limiter-rps", 2, "rate limiter max requests per second")
	flag.IntVar(&cfg.limiter.burst, "limiter-burst", 4, "rate limiter max burst")
	flag.BoolVar(&cfg.limiter.enabled, "limiter-enabled", true, "enable rate limiter")

	flag.StringVar(&cfg.smtp.host, "smtp-host", "sandbox.smtp.mailtrap.io", "smtp host")
	flag.IntVar(&cfg.smtp.port, "smtp-port", 2525, "smtp port")
	flag.StringVar(&cfg.smtp.username, "smtp-username", "3d6b7435ce08c3", "smtp username")
	flag.StringVar(&cfg.smtp.password, "smtp-password", "20910f067fc163", "smtp password")
	flag.StringVar(&cfg.smtp.sender, "smtp-sender", "tuck <no-reply@tuck.loveless.dev>", "smtp sender")

	flag.Func("cors-trusted-origins", "trusted CORS origins (space separated)", func(val string) error {
		cfg.cors.trustedOrigins = strings.Fields(val)
		return nil
	})

	displayVersion := flag.Bool("version", false, "display version and exit")

	flag.Parse()

	if *displayVersion {
		fmt.Printf("version:\t%s\n", version)
		os.Exit(0)
	}

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

	templateCache, err := newTemplateCache()
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}

	mailer, err := mailer.New(cfg.smtp.host, cfg.smtp.port, cfg.smtp.username, cfg.smtp.password, cfg.smtp.sender)
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}

	expvar.NewString("version").Set(version)

	expvar.Publish("goroutines", expvar.Func(func() any {
		return runtime.NumGoroutine()
	}))

	expvar.Publish("database", expvar.Func(func() any {
		return db.Stats()
	}))

	expvar.Publish("timestamp", expvar.Func(func() any {
		return time.Now().Unix()
	}))

	// declare an instance of the application struct, containing the config
	// struct, logger, and models
	app := &application{
		config:        cfg,
		logger:        logger,
		models:        data.NewModels(db),
		mailer:        mailer,
		templateCache: templateCache,
	}

	err = app.serve()
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}
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
