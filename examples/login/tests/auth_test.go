/*
Tests for the login example's RS256 auth middleware.

Untagged: no database and no server of its own, so `go test ./...` runs them
on every commit. The keypair is generated in TestMain rather than committed,
which also means these tests can't accidentally pass against a key that only
exists on one machine.
*/
package tests

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Binh-2060/go-application-template/examples/login/controllers"
	"github.com/Binh-2060/go-application-template/examples/login/middlewares"
	loginroutes "github.com/Binh-2060/go-application-template/examples/login/routes"
	"github.com/Binh-2060/go-application-template/examples/login/services"
	"github.com/Binh-2060/go-application-template/internal/api/validators"
	appjwt "github.com/Binh-2060/go-application-template/pkg/jwt"
	"github.com/gofiber/fiber/v3"
)

// signer mints tokens the middleware should accept; otherSigner mints tokens
// signed by a valid-but-wrong key, which it must reject.
var signer, otherSigner *appjwt.RSAManager

func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "login-jwt")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(dir)

	privPath, pubPath := writeKeypair(dir, "jwt")
	os.Setenv("JWT_RSA_PRIVATE_KEY_PATH", privPath)
	os.Setenv("JWT_RSA_PUBLIC_KEY_PATH", pubPath)
	os.Setenv("JWT_ISSUER", "login-example-tests")

	validators.Init()
	if err := services.Init(); err != nil {
		panic(err)
	}
	if err := middlewares.Init(); err != nil {
		panic(err)
	}

	cfg, err := appjwt.RSAConfigFromEnv()
	if err != nil {
		panic(err)
	}
	if signer, err = appjwt.NewRSAManager(cfg); err != nil {
		panic(err)
	}

	otherPriv, otherPub := writeKeypair(dir, "other")
	otherCfg := cfg
	otherCfg.PrivateKeyPEM, otherCfg.PublicKeyPEM = readFile(otherPriv), readFile(otherPub)
	if otherSigner, err = appjwt.NewRSAManager(otherCfg); err != nil {
		panic(err)
	}

	code := m.Run()
	os.RemoveAll(dir)
	os.Exit(code)
}

// newApp mounts the feature exactly as internal/api/routes would, so the
// tests exercise the real SetAuthRoute wiring rather than a handler chain
// assembled here that could drift from it.
func newApp() *fiber.App {
	app := fiber.New()
	loginroutes.SetAuthRoute(app.Group("/login"))
	return app
}

func TestRequireAuthAcceptsValidTokenAndExposesSubject(t *testing.T) {
	token, err := signer.SignWithTTL("user-42", time.Minute)
	if err != nil {
		t.Fatal(err)
	}

	res := do(t, newApp(), http.MethodGet, "/login/me", "Bearer "+token)
	if res.status != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body %s)", res.status, res.body)
	}
	if got := res.items["user_id"]; got != "user-42" {
		t.Errorf("user_id = %q, want %q", got, "user-42")
	}
}

// RFC 6750 makes the auth scheme case-insensitive, so a client sending
// "bearer" is not sending a malformed request.
func TestRequireAuthAcceptsLowercaseScheme(t *testing.T) {
	token, err := signer.SignWithTTL("user-42", time.Minute)
	if err != nil {
		t.Fatal(err)
	}

	res := do(t, newApp(), http.MethodGet, "/login/me", "bearer "+token)
	if res.status != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body %s)", res.status, res.body)
	}
}

func TestRequireAuthRejects(t *testing.T) {
	expired, err := signer.SignWithTTL("user-42", -time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	forged, err := otherSigner.SignWithTTL("user-42", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	valid, err := signer.SignWithTTL("user-42", time.Minute)
	if err != nil {
		t.Fatal(err)
	}

	// No leading-whitespace case: net/http trims header values, so " Bearer x"
	// never reaches the middleware un-trimmed and the case would test the
	// transport rather than RequireAuth.
	cases := map[string]string{
		"no header":        "",
		"scheme only":      "Bearer",
		"empty token":      "Bearer ",
		"wrong scheme":     "Basic " + valid,
		"no scheme":        valid,
		"garbage token":    "Bearer not-a-jwt",
		"expired token":    "Bearer " + expired,
		"signed other key": "Bearer " + forged,
		"tampered payload": "Bearer " + tamper(valid),
	}

	app := newApp()
	for name, header := range cases {
		t.Run(name, func(t *testing.T) {
			res := do(t, app, http.MethodGet, "/login/me", header)
			if res.status != http.StatusUnauthorized {
				t.Fatalf("status = %d, want 401 (body %s)", res.status, res.body)
			}
		})
	}
}

/*
Regression guard for the fail-open case: a handler reading UserID must 401
when RequireAuth was never mounted, not serve a 200 with an empty user ID.

Mounts controllers.Me bare — the mistake this is guarding against.
*/
func TestProtectedHandlerMountedWithoutMiddlewareRejects(t *testing.T) {
	app := fiber.New()
	app.Get("/me", controllers.Me)

	res := do(t, app, http.MethodGet, "/me", "")
	if res.status != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401 (body %s)", res.status, res.body)
	}
}

// End to end: the token /login hands out is one RequireAuth accepts. Catches
// an issuer or algorithm mismatch between the signing and verifying halves,
// which are now two separately-built managers.
func TestLoginIssuesTokenAcceptedByRequireAuth(t *testing.T) {
	app := newApp()

	req := httptest.NewRequest(http.MethodPost, "/login/",
		strings.NewReader(`{"email":"demo@example.com","password":"password123"}`))
	req.Header.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)
	res := send(t, app, req)
	if res.status != http.StatusOK {
		t.Fatalf("login status = %d, want 200 (body %s)", res.status, res.body)
	}

	token := res.items["access_token"]
	if token == "" {
		t.Fatalf("login returned no access_token (body %s)", res.body)
	}

	me := do(t, app, http.MethodGet, "/login/me", "Bearer "+token)
	if me.status != http.StatusOK {
		t.Fatalf("/me status = %d, want 200 (body %s)", me.status, me.body)
	}
}

func TestLoginRejectsBadCredentials(t *testing.T) {
	cases := map[string]string{
		"wrong password": `{"email":"demo@example.com","password":"wrongpassword"}`,
		"unknown email":  `{"email":"nobody@example.com","password":"password123"}`,
	}

	app := newApp()
	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/login/", strings.NewReader(body))
			req.Header.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)

			res := send(t, app, req)
			if res.status != http.StatusUnauthorized {
				t.Fatalf("status = %d, want 401 (body %s)", res.status, res.body)
			}
		})
	}
}

// --- helpers ---

type result struct {
	status int
	body   string
	items  map[string]string
}

func do(t *testing.T, app *fiber.App, method, path, authHeader string) result {
	t.Helper()

	req := httptest.NewRequest(method, path, nil)
	if authHeader != "" {
		req.Header.Set(fiber.HeaderAuthorization, authHeader)
	}
	return send(t, app, req)
}

func send(t *testing.T, app *fiber.App, req *http.Request) result {
	t.Helper()

	res, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}

	// Only the string fields of "items" are needed here, and the envelope
	// carries mixed types, so decode loosely rather than importing the
	// response struct.
	var envelope struct {
		Items map[string]any `json:"items"`
	}
	items := map[string]string{}
	if json.Unmarshal(body, &envelope) == nil {
		for k, v := range envelope.Items {
			if s, ok := v.(string); ok {
				items[k] = s
			}
		}
	}

	return result{status: res.StatusCode, body: string(body), items: items}
}

func writeKeypair(dir, name string) (privPath, pubPath string) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		panic(err)
	}

	privPath = filepath.Join(dir, name+"_private.pem")
	privPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	})
	if err := os.WriteFile(privPath, privPEM, 0o600); err != nil {
		panic(err)
	}

	der, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		panic(err)
	}
	pubPath = filepath.Join(dir, name+"_public.pem")
	pubPEM := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der})
	if err := os.WriteFile(pubPath, pubPEM, 0o600); err != nil {
		panic(err)
	}

	return privPath, pubPath
}

func readFile(path string) string {
	b, err := os.ReadFile(path)
	if err != nil {
		panic(err)
	}
	return string(b)
}

// tamper flips a character in the token's payload segment, leaving the
// signature intact — the signature no longer matches the payload.
func tamper(token string) string {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		panic("not a JWT")
	}
	payload := []byte(parts[1])
	if payload[0] == 'A' {
		payload[0] = 'B'
	} else {
		payload[0] = 'A'
	}
	parts[1] = string(payload)
	return strings.Join(parts, ".")
}
