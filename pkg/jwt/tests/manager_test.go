package tests

import (
	"errors"
	"strings"
	"testing"
	"time"

	appjwt "github.com/Binh-2060/go-application-template/pkg/jwt"
	gojwt "github.com/golang-jwt/jwt/v5"
)

/*
Tests for the HS256 Manager.

The happy path is one test; the rest are the ways a token that should be
rejected could get accepted. That ratio is the point — a JWT package that only
proves it can verify its own tokens proves almost nothing.
*/

func TestManagerSignVerifyRoundTrip(t *testing.T) {
	m := newHMACManager(t)

	token, err := m.Sign("user-123")
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}

	claims, err := m.Verify(token)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}

	if claims.Subject != "user-123" {
		t.Errorf("Subject = %q, want %q", claims.Subject, "user-123")
	}
	if claims.Issuer != testIssuer {
		t.Errorf("Issuer = %q, want %q", claims.Issuer, testIssuer)
	}
	if claims.ExpiresAt == nil || claims.IssuedAt == nil {
		t.Fatal("token must carry both iat and exp")
	}
	if got := claims.ExpiresAt.Sub(claims.IssuedAt.Time); got != time.Hour {
		t.Errorf("exp - iat = %s, want the configured TTL of 1h", got)
	}
}

// SignWithTTL is what lets one Manager mint both short access tokens and long
// refresh tokens, so the caller's TTL must win over the configured one.
func TestManagerSignWithTTLOverridesConfiguredTTL(t *testing.T) {
	m := newHMACManager(t)

	token, err := m.SignWithTTL("user-123", 5*time.Minute)
	if err != nil {
		t.Fatalf("SignWithTTL: %v", err)
	}

	claims, err := m.Verify(token)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}

	if got := claims.ExpiresAt.Sub(claims.IssuedAt.Time); got != 5*time.Minute {
		t.Errorf("exp - iat = %s, want 5m", got)
	}
}

func TestNewManagerRejectsEmptySecret(t *testing.T) {
	cfg := hmacConfig()
	cfg.Secret = ""

	if _, err := appjwt.NewManager(cfg); !errors.Is(err, appjwt.ErrEmptySecret) {
		t.Fatalf("NewManager with empty secret: err = %v, want ErrEmptySecret", err)
	}
}

/*
A short HMAC secret is recoverable offline from a single captured token, which
makes every token forgeable. RFC 7518 §3.2 requires >= 256 bits for HS256, so
a plausible-looking dev secret must fail at construction rather than at audit.
*/
func TestNewManagerRejectsShortSecret(t *testing.T) {
	cfg := hmacConfig()
	cfg.Secret = "dev-secret" // 10 bytes

	if _, err := appjwt.NewManager(cfg); !errors.Is(err, appjwt.ErrWeakSecret) {
		t.Fatalf("NewManager with short secret: err = %v, want ErrWeakSecret", err)
	}
}

func TestManagerVerifyRejectsExpiredToken(t *testing.T) {
	m := newHMACManager(t)

	// Well past the leeway window, so this is expiry and not skew tolerance.
	token, err := m.SignWithTTL("user-123", -time.Hour)
	if err != nil {
		t.Fatalf("SignWithTTL: %v", err)
	}

	_, err = m.Verify(token)
	if !errors.Is(err, appjwt.ErrExpiredToken) {
		t.Fatalf("Verify expired token: err = %v, want ErrExpiredToken", err)
	}
}

/*
An expired-by-seconds token must still verify: hosts disagree about the time,
and without leeway a token minted on a clock a few seconds ahead is rejected
by every other machine in the fleet.
*/
func TestManagerVerifyToleratesClockSkewWithinLeeway(t *testing.T) {
	m := newHMACManager(t)

	token, err := m.SignWithTTL("user-123", -5*time.Second)
	if err != nil {
		t.Fatalf("SignWithTTL: %v", err)
	}

	if _, err := m.Verify(token); err != nil {
		t.Fatalf("Verify token expired 5s ago: %v, want it accepted within the 30s leeway", err)
	}
}

// A token from another issuer is a token from another system's key management.
func TestManagerVerifyRejectsForeignIssuer(t *testing.T) {
	cfg := hmacConfig()
	cfg.Issuer = "some-other-service"
	other, err := appjwt.NewManager(cfg)
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}

	token, err := other.Sign("user-123")
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}

	if _, err := newHMACManager(t).Verify(token); !errors.Is(err, appjwt.ErrInvalidToken) {
		t.Fatalf("Verify foreign-issuer token: err = %v, want ErrInvalidToken", err)
	}
}

func TestManagerVerifyRejectsForeignSecret(t *testing.T) {
	cfg := hmacConfig()
	cfg.Secret = strings.Repeat("z", 32)
	other, err := appjwt.NewManager(cfg)
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}

	token, err := other.Sign("user-123")
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}

	if _, err := newHMACManager(t).Verify(token); !errors.Is(err, appjwt.ErrInvalidToken) {
		t.Fatalf("Verify token signed with another secret: err = %v, want ErrInvalidToken", err)
	}
}

func TestManagerVerifyRejectsTamperedPayload(t *testing.T) {
	m := newHMACManager(t)

	token, err := m.Sign("user-123")
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}

	// Swap the payload segment for one claiming a different subject, keeping
	// the original header and signature.
	forged, err := m.Sign("admin")
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}
	parts, forgedParts := strings.Split(token, "."), strings.Split(forged, ".")
	spliced := parts[0] + "." + forgedParts[1] + "." + parts[2]

	if _, err := m.Verify(spliced); !errors.Is(err, appjwt.ErrInvalidToken) {
		t.Fatalf("Verify spliced token: err = %v, want ErrInvalidToken", err)
	}
}

/*
The alg:"none" attack: a token whose header declares no signature at all. A
verifier that trusts the header over its own configuration accepts it with an
empty signature, which means anyone can mint any claims.
*/
func TestManagerVerifyRejectsAlgNone(t *testing.T) {
	claims := gojwt.RegisteredClaims{
		Issuer:    testIssuer,
		Subject:   "admin",
		IssuedAt:  gojwt.NewNumericDate(time.Now()),
		ExpiresAt: gojwt.NewNumericDate(time.Now().Add(time.Hour)),
	}
	token, err := gojwt.NewWithClaims(gojwt.SigningMethodNone, claims).
		SignedString(gojwt.UnsafeAllowNoneSignatureType)
	if err != nil {
		t.Fatalf("build alg:none token: %v", err)
	}

	if _, err := newHMACManager(t).Verify(token); !errors.Is(err, appjwt.ErrInvalidToken) {
		t.Fatalf("Verify alg:none token: err = %v, want ErrInvalidToken", err)
	}
}

/*
An RS256 token handed to the HS256 Manager must be rejected on the algorithm,
before the keyfunc's secret is used for anything.
*/
func TestManagerVerifyRejectsRSASignedToken(t *testing.T) {
	token, err := newRSAManager(t).Sign("admin")
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}

	_, err = newHMACManager(t).Verify(token)
	if !errors.Is(err, appjwt.ErrInvalidToken) {
		t.Fatalf("Verify RS256 token with HMAC manager: err = %v, want ErrInvalidToken", err)
	}
	// The pin must be what rejected it — see the RSA counterpart for why
	// checking only for rejection is not enough.
	if !errors.Is(err, appjwt.ErrUnexpectedSigningMethod) {
		t.Errorf("rejection should come from the HMAC algorithm pin, got %v", err)
	}
}

/*
Verify must not tell a caller why a token failed beyond expired-or-not: the
distinction between "bad signature" and "wrong issuer" is an oracle. The cause
is still wrapped underneath for server-side logs.
*/
func TestVerifyErrorsStayCoarseButKeepTheCause(t *testing.T) {
	m := newHMACManager(t)

	token, err := m.SignWithTTL("user-123", -time.Hour)
	if err != nil {
		t.Fatalf("SignWithTTL: %v", err)
	}

	_, err = m.Verify(token)
	if !errors.Is(err, gojwt.ErrTokenExpired) {
		t.Errorf("underlying library error should stay wrapped, got %v", err)
	}
	if errors.Is(err, appjwt.ErrInvalidToken) {
		t.Error("an expired token must not also match ErrInvalidToken; callers switch on these")
	}
}
