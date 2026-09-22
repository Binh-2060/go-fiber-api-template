/*
Package tests holds every test for pkg/jwt.

The tests live outside the package they exercise, so they see the same exported
API a caller would — and, more to the point, the same API an attacker's forged
token meets.

No build tag and no TestMain: this package is pure crypto and clock arithmetic,
so it runs on every `go test ./...` with no database and no .env. That is
deliberate. The properties asserted here (algorithm pinning, issuer checking,
key-strength floors) are exactly the ones that fail silently when someone edits
a keyfunc, so they must not sit behind a tag that CI can skip.

Keys are generated per-test rather than checked in as fixtures: a committed
private key is a key that leaks, even a throwaway one.
*/
package tests

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"os"
	"testing"
	"time"

	appjwt "github.com/Binh-2060/go-application-template/pkg/jwt"
)

// A secret long enough to clear the minSecretLen floor in pkg/jwt.
const testSecret = "0123456789abcdef0123456789abcdef"

const testIssuer = "test-issuer"

func hmacConfig() appjwt.Config {
	return appjwt.Config{Secret: testSecret, Issuer: testIssuer, TTL: time.Hour}
}

func newHMACManager(t *testing.T) *appjwt.Manager {
	t.Helper()

	m, err := appjwt.NewManager(hmacConfig())
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}
	return m
}

// Generate an RSA keypair of the given size and return it PEM-encoded, in the
// same shape RSAConfigFromEnv would have read off disk.
func rsaKeyPairPEM(t *testing.T, bits int) (privatePEM, publicPEM string) {
	t.Helper()

	key, err := rsa.GenerateKey(rand.Reader, bits)
	if err != nil {
		t.Fatalf("generate %d-bit key: %v", bits, err)
	}

	privatePEM = string(pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	}))

	pubDER, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		t.Fatalf("marshal public key: %v", err)
	}
	publicPEM = string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubDER}))

	return privatePEM, publicPEM
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()

	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func rsaConfig(t *testing.T) appjwt.RSAConfig {
	t.Helper()

	priv, pub := rsaKeyPairPEM(t, 2048)
	return appjwt.RSAConfig{
		PrivateKeyPEM: priv,
		PublicKeyPEM:  pub,
		Issuer:        testIssuer,
		TTL:           time.Hour,
	}
}

func newRSAManager(t *testing.T) *appjwt.RSAManager {
	t.Helper()

	m, err := appjwt.NewRSAManager(rsaConfig(t))
	if err != nil {
		t.Fatalf("NewRSAManager: %v", err)
	}
	return m
}
