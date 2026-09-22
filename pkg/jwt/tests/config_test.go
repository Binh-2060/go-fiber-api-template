package tests

import (
	"testing"
	"time"

	appjwt "github.com/Binh-2060/go-application-template/pkg/jwt"
)

/*
Tests for environment loading.

The interesting case is not the happy path but the typo: a TTL the operator
believed they set. These use t.Setenv, so they must not run in parallel.
*/

func TestConfigFromEnvReadsAllVars(t *testing.T) {
	t.Setenv("JWT_SECRET", testSecret)
	t.Setenv("JWT_ISSUER", "billing-api")
	t.Setenv("JWT_TTL", "15m")

	cfg, err := appjwt.ConfigFromEnv()
	if err != nil {
		t.Fatalf("ConfigFromEnv: %v", err)
	}

	if cfg.Secret != testSecret {
		t.Errorf("Secret = %q, want the value of JWT_SECRET", cfg.Secret)
	}
	if cfg.Issuer != "billing-api" {
		t.Errorf("Issuer = %q, want %q", cfg.Issuer, "billing-api")
	}
	if cfg.TTL != 15*time.Minute {
		t.Errorf("TTL = %s, want 15m", cfg.TTL)
	}
}

// JWT_ISSUER falls back to API_NAME so a service doesn't have to name itself
// twice, and to a package default when neither is set.
func TestConfigFromEnvIssuerFallsBackToAPIName(t *testing.T) {
	t.Setenv("JWT_ISSUER", "")
	t.Setenv("API_NAME", "orders-api")
	t.Setenv("JWT_TTL", "")

	cfg, err := appjwt.ConfigFromEnv()
	if err != nil {
		t.Fatalf("ConfigFromEnv: %v", err)
	}

	if cfg.Issuer != "orders-api" {
		t.Errorf("Issuer = %q, want it to fall back to API_NAME", cfg.Issuer)
	}
}

// An unset JWT_TTL is a real default, not a misconfiguration.
func TestConfigFromEnvUnsetTTLUsesDefault(t *testing.T) {
	t.Setenv("JWT_TTL", "")

	cfg, err := appjwt.ConfigFromEnv()
	if err != nil {
		t.Fatalf("ConfigFromEnv with unset TTL: %v", err)
	}

	if cfg.TTL != 24*time.Hour {
		t.Errorf("TTL = %s, want the 24h default declared in config.go", cfg.TTL)
	}
}

/*
A set-but-unparseable TTL must fail loudly.

"15" means fifteen of nothing; time.ParseDuration rejects it. Falling back to
the 24h default here would hand out day-long tokens to a service that asked
for fifteen minutes, and no log line would ever mention it.
*/
func TestConfigFromEnvRejectsMalformedTTL(t *testing.T) {
	for _, raw := range []string{"15", "1hour", "abc"} {
		t.Run(raw, func(t *testing.T) {
			t.Setenv("JWT_TTL", raw)

			if _, err := appjwt.ConfigFromEnv(); err == nil {
				t.Fatalf("ConfigFromEnv with JWT_TTL=%q returned no error", raw)
			}
		})
	}
}

// A zero or negative TTL mints tokens that are already expired.
func TestConfigFromEnvRejectsNonPositiveTTL(t *testing.T) {
	for _, raw := range []string{"0s", "-5m"} {
		t.Run(raw, func(t *testing.T) {
			t.Setenv("JWT_TTL", raw)

			if _, err := appjwt.ConfigFromEnv(); err == nil {
				t.Fatalf("ConfigFromEnv with JWT_TTL=%q returned no error", raw)
			}
		})
	}
}

// The same validation must apply on the RSA path; it reads the same var.
func TestRSAConfigFromEnvRejectsMalformedTTL(t *testing.T) {
	t.Setenv("JWT_TTL", "15")

	if _, err := appjwt.RSAConfigFromEnv(); err == nil {
		t.Fatal("RSAConfigFromEnv with JWT_TTL=15 returned no error")
	}
}

// Key paths are read from disk, so a wrong path is a startup error rather than
// a nil key discovered on the first request.
func TestRSAConfigFromEnvReportsMissingKeyFile(t *testing.T) {
	t.Setenv("JWT_TTL", "")
	t.Setenv("JWT_RSA_PUBLIC_KEY_PATH", "/nonexistent/public.pem")

	if _, err := appjwt.RSAConfigFromEnv(); err == nil {
		t.Fatal("RSAConfigFromEnv with a missing key file returned no error")
	}
}

func TestRSAConfigFromEnvReadsKeysFromDisk(t *testing.T) {
	priv, pub := rsaKeyPairPEM(t, 2048)
	dir := t.TempDir()
	privPath, pubPath := dir+"/private.pem", dir+"/public.pem"

	writeFile(t, privPath, priv)
	writeFile(t, pubPath, pub)

	t.Setenv("JWT_TTL", "30m")
	t.Setenv("JWT_ISSUER", "keys-from-disk")
	t.Setenv("JWT_RSA_PRIVATE_KEY_PATH", privPath)
	t.Setenv("JWT_RSA_PUBLIC_KEY_PATH", pubPath)

	cfg, err := appjwt.RSAConfigFromEnv()
	if err != nil {
		t.Fatalf("RSAConfigFromEnv: %v", err)
	}

	// The proof the PEMs survived the round trip is that a manager builds and
	// signs, not that the strings compare equal.
	m, err := appjwt.NewRSAManager(cfg)
	if err != nil {
		t.Fatalf("NewRSAManager from env config: %v", err)
	}
	token, err := m.Sign("user-123")
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}
	if _, err := m.Verify(token); err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if cfg.TTL != 30*time.Minute {
		t.Errorf("TTL = %s, want 30m", cfg.TTL)
	}
}
