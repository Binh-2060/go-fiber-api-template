package db

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/url"
	"os"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// pool is the process-wide connection pool. It is safe for concurrent use by
// multiple goroutines, so the whole app shares this single instance.
var pool *pgxpool.Pool

/*
Config holds every knob the Postgres pool needs.

Populated from the environment by ConfigFromEnv, but exposed so tests can build
one directly without touching os.Getenv.
*/
type Config struct {
	Host     string
	Port     string
	User     string
	Password string
	Database string
	SSLMode  string

	// URL, when set, wins over the discrete fields above. Useful for hosted
	// Postgres providers that hand you a single connection string.
	URL string

	MaxConns        int32
	MinConns        int32
	MaxConnLifetime time.Duration
	MaxConnIdleTime time.Duration

	// ConnectTimeout bounds the initial dial + ping during Init.
	ConnectTimeout time.Duration
}

/*
Read pool configuration from environment variables.

Recognised vars: DATABASE_URL, DB_HOST, DB_PORT, DB_USER, DB_PASSWORD, DB_NAME,
DB_SSLMODE, DB_MAX_CONNS, DB_MIN_CONNS, DB_MAX_CONN_LIFETIME,
DB_MAX_CONN_IDLE_TIME, DB_CONNECT_TIMEOUT.
*/
func ConfigFromEnv() Config {
	return Config{
		URL:      os.Getenv("DATABASE_URL"),
		Host:     envOr("DB_HOST", "localhost"),
		Port:     envOr("DB_PORT", "5432"),
		User:     envOr("DB_USER", "postgres"),
		Password: os.Getenv("DB_PASSWORD"),
		Database: envOr("DB_NAME", "postgres"),
		SSLMode:  envOr("DB_SSLMODE", "disable"),

		MaxConns:        int32(envIntOr("DB_MAX_CONNS", 25)),
		MinConns:        int32(envIntOr("DB_MIN_CONNS", 2)),
		MaxConnLifetime: envDurationOr("DB_MAX_CONN_LIFETIME", time.Hour),
		MaxConnIdleTime: envDurationOr("DB_MAX_CONN_IDLE_TIME", 30*time.Minute),
		ConnectTimeout:  envDurationOr("DB_CONNECT_TIMEOUT", 10*time.Second),
	}
}

/*
Build a libpq-style connection string.

The password is URL-encoded rather than concatenated, so passwords containing
'@', '/' or ':' do not corrupt the DSN.
*/
func (c Config) DSN() string {
	if c.URL != "" {
		return c.URL
	}

	u := &url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(c.User, c.Password),
		Host:   c.Host + ":" + c.Port,
		Path:   "/" + c.Database,
	}

	q := u.Query()
	q.Set("sslmode", c.SSLMode)
	u.RawQuery = q.Encode()

	return u.String()
}

/*
Open the pool and verify it with a ping.

Call once from main.go before serving traffic. pgxpool connects lazily, so the
ping is what turns a bad host/password into a startup failure instead of a
surprise on the first request.
*/
func Init(ctx context.Context, cfg Config) error {
	if pool != nil {
		return errors.New("db: already initialised")
	}

	poolCfg, err := pgxpool.ParseConfig(cfg.DSN())
	if err != nil {
		return fmt.Errorf("db: parse config: %w", err)
	}

	poolCfg.MaxConns = cfg.MaxConns
	poolCfg.MinConns = cfg.MinConns
	poolCfg.MaxConnLifetime = cfg.MaxConnLifetime
	poolCfg.MaxConnIdleTime = cfg.MaxConnIdleTime

	ctx, cancel := context.WithTimeout(ctx, cfg.ConnectTimeout)
	defer cancel()

	p, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return fmt.Errorf("db: create pool: %w", err)
	}

	if err := p.Ping(ctx); err != nil {
		p.Close()
		return fmt.Errorf("db: ping: %w", err)
	}

	log.Println("connection db success")
	pool = p
	return nil
}

/*
Return the shared pool.

Panics if Init has not run — a nil pool is a wiring bug, and failing loudly at
the first query beats a nil-pointer dereference deep in a handler.
*/
func Pool() *pgxpool.Pool {
	if pool == nil {
		panic("db: Pool() called before Init()")
	}
	return pool
}

/*
Report whether the database is reachable. Intended for /healthz.
*/
func Ping(ctx context.Context) error {
	if pool == nil {
		return errors.New("db: not initialised")
	}
	return pool.Ping(ctx)
}

/*
Close the pool. Safe to call even if Init never succeeded.

Blocks until in-flight queries finish, so call it during graceful shutdown
after the HTTP server has stopped accepting connections.
*/
func Close() {
	if pool != nil {
		pool.Close()
		pool = nil
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envIntOr(key string, fallback int) int {
	v, err := strconv.Atoi(os.Getenv(key))
	if err != nil {
		return fallback
	}
	return v
}

func envDurationOr(key string, fallback time.Duration) time.Duration {
	v, err := time.ParseDuration(os.Getenv(key))
	if err != nil {
		return fallback
	}
	return v
}
