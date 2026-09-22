package jwt

import (
	"fmt"
	"os"
	"time"
)

const (
	defaultTTL    = 24 * time.Hour
	defaultIssuer = "go-fiber-api-template"
)

// Config holds the knobs needed to construct a Manager.
type Config struct {
	Secret string
	Issuer string
	TTL    time.Duration
}

// RSAConfig holds the knobs needed to construct an RSAManager.
type RSAConfig struct {
	PrivateKeyPEM string // empty builds a verify-only Manager
	PublicKeyPEM  string
	Issuer        string
	TTL           time.Duration
}

/*
Read JWT configuration from environment variables.

Recognised vars: JWT_SECRET (required, no default — Manager rejects a missing
or too-short secret), JWT_ISSUER (default API_NAME, falling back to
"go-fiber-api-template"), JWT_TTL (default 24h).

Returns an error for a malformed JWT_TTL rather than falling back, so a typo
can't silently widen a token's lifetime.
*/
func ConfigFromEnv() (Config, error) {
	ttl, err := envDurationOr("JWT_TTL", defaultTTL)
	if err != nil {
		return Config{}, err
	}

	return Config{
		Secret: os.Getenv("JWT_SECRET"),
		Issuer: envOr("JWT_ISSUER", envOr("API_NAME", defaultIssuer)),
		TTL:    ttl,
	}, nil
}

/*
Read RSA JWT configuration from environment variables.

Recognised vars: JWT_RSA_PRIVATE_KEY_PATH (path to a PEM-encoded RSA private
key, optional — omit to build a verify-only Manager), JWT_RSA_PUBLIC_KEY_PATH
(path to the matching PEM-encoded public key, required), JWT_ISSUER, JWT_TTL
(same defaults as ConfigFromEnv).

Keys are read from files rather than inlined in .env so PEM material — which
spans multiple lines — doesn't need escaping into a single env var, and
doesn't get dumped by `env`/process-listing tools the way an env var would.
*/
func RSAConfigFromEnv() (RSAConfig, error) {
	ttl, err := envDurationOr("JWT_TTL", defaultTTL)
	if err != nil {
		return RSAConfig{}, err
	}

	cfg := RSAConfig{
		Issuer: envOr("JWT_ISSUER", envOr("API_NAME", defaultIssuer)),
		TTL:    ttl,
	}

	if path := os.Getenv("JWT_RSA_PRIVATE_KEY_PATH"); path != "" {
		b, err := os.ReadFile(path)
		if err != nil {
			return RSAConfig{}, fmt.Errorf("jwt: read private key: %w", err)
		}
		cfg.PrivateKeyPEM = string(b)
	}

	if path := os.Getenv("JWT_RSA_PUBLIC_KEY_PATH"); path != "" {
		b, err := os.ReadFile(path)
		if err != nil {
			return RSAConfig{}, fmt.Errorf("jwt: read public key: %w", err)
		}
		cfg.PublicKeyPEM = string(b)
	}

	return cfg, nil
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

/*
Read a duration from key, falling back only when the var is unset.

A set-but-unparseable value is an error, not a fallback: JWT_TTL=24 (no unit)
would otherwise silently mint 24-hour tokens for a service that asked for 24
minutes, and nothing would say so.
*/
func envDurationOr(key string, fallback time.Duration) (time.Duration, error) {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback, nil
	}

	v, err := time.ParseDuration(raw)
	if err != nil {
		return 0, fmt.Errorf("jwt: invalid %s %q: %w", key, raw, err)
	}
	if v <= 0 {
		return 0, fmt.Errorf("jwt: %s must be positive, got %s", key, v)
	}
	return v, nil
}
