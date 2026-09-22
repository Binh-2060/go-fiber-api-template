package tests

import (
	"errors"
	"testing"
	"time"

	appjwt "github.com/Binh-2060/go-application-template/pkg/jwt"
	gojwt "github.com/golang-jwt/jwt/v5"
)

/*
Tests for the RS256 RSAManager.

Two properties here have no counterpart in the HS256 tests and carry most of
the value: the verify-only manager really cannot sign, and a public key really
cannot be replayed as an HMAC secret.
*/

func TestRSAManagerSignVerifyRoundTrip(t *testing.T) {
	m := newRSAManager(t)

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
}

func TestNewRSAManagerRequiresPublicKey(t *testing.T) {
	cfg := rsaConfig(t)
	cfg.PublicKeyPEM = ""

	if _, err := appjwt.NewRSAManager(cfg); !errors.Is(err, appjwt.ErrMissingPublicKey) {
		t.Fatalf("NewRSAManager without public key: err = %v, want ErrMissingPublicKey", err)
	}
}

/*
The verify-only mode is the reason to prefer RS256 over HS256 across services:
a downstream service gets the public key only, so compromising it yields no
ability to mint tokens. If Sign worked here, that separation would be a
comment rather than a property.
*/
func TestVerifyOnlyRSAManagerCannotSign(t *testing.T) {
	full := rsaConfig(t)

	verifyOnly, err := appjwt.NewRSAManager(appjwt.RSAConfig{
		PublicKeyPEM: full.PublicKeyPEM,
		Issuer:       full.Issuer,
		TTL:          full.TTL,
	})
	if err != nil {
		t.Fatalf("NewRSAManager: %v", err)
	}

	if _, err := verifyOnly.Sign("user-123"); !errors.Is(err, appjwt.ErrMissingPrivateKey) {
		t.Fatalf("Sign on verify-only manager: err = %v, want ErrMissingPrivateKey", err)
	}
	if _, err := verifyOnly.SignWithTTL("user-123", time.Minute); !errors.Is(err, appjwt.ErrMissingPrivateKey) {
		t.Fatalf("SignWithTTL on verify-only manager: err = %v, want ErrMissingPrivateKey", err)
	}

	// It must still verify tokens the signing side issued.
	signer, err := appjwt.NewRSAManager(full)
	if err != nil {
		t.Fatalf("NewRSAManager: %v", err)
	}
	token, err := signer.Sign("user-123")
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}
	if _, err := verifyOnly.Verify(token); err != nil {
		t.Fatalf("verify-only manager must still verify: %v", err)
	}
}

// NIST deprecated 1024-bit RSA in 2013; the floor is checked against the
// parsed modulus, so an undersized key can't slip through on config alone.
func TestNewRSAManagerRejectsUndersizedKey(t *testing.T) {
	priv, pub := rsaKeyPairPEM(t, 1024)

	_, err := appjwt.NewRSAManager(appjwt.RSAConfig{
		PrivateKeyPEM: priv,
		PublicKeyPEM:  pub,
		Issuer:        testIssuer,
		TTL:           time.Hour,
	})
	if err == nil {
		t.Fatal("NewRSAManager accepted a 1024-bit key")
	}
}

func TestNewRSAManagerRejectsMalformedPEM(t *testing.T) {
	cfg := rsaConfig(t)
	cfg.PublicKeyPEM = "-----BEGIN PUBLIC KEY-----\nnot base64\n-----END PUBLIC KEY-----\n"

	if _, err := appjwt.NewRSAManager(cfg); err == nil {
		t.Fatal("NewRSAManager accepted a malformed public key")
	}
}

/*
The classic RS256 -> HS256 confusion attack.

The public key is, by definition, public. If the verifier honours the token's
own "alg" header, an attacker re-signs their claims with HS256 using the PEM
text of the public key as the HMAC secret, and the verifier — reaching for
"the key" — validates it. Pinning the method to RSA is what stops this.
*/
func TestRSAManagerVerifyRejectsHMACSignedTokenUsingPublicKeyAsSecret(t *testing.T) {
	cfg := rsaConfig(t)

	m, err := appjwt.NewRSAManager(cfg)
	if err != nil {
		t.Fatalf("NewRSAManager: %v", err)
	}

	claims := gojwt.RegisteredClaims{
		Issuer:    testIssuer,
		Subject:   "admin",
		IssuedAt:  gojwt.NewNumericDate(time.Now()),
		ExpiresAt: gojwt.NewNumericDate(time.Now().Add(time.Hour)),
	}
	forged, err := gojwt.NewWithClaims(gojwt.SigningMethodHS256, claims).
		SignedString([]byte(cfg.PublicKeyPEM))
	if err != nil {
		t.Fatalf("build forged HS256 token: %v", err)
	}

	_, err = m.Verify(forged)
	if !errors.Is(err, appjwt.ErrInvalidToken) {
		t.Fatalf("Verify HS256-forged token: err = %v, want ErrInvalidToken", err)
	}

	/*
		Assert the rejection came from the algorithm pin specifically.

		Without this the test passes even with the pin deleted, because
		golang-jwt would then hand an *rsa.PublicKey to the HMAC verifier and
		fail on the key type instead. That is a real second line of defence,
		but it is the library's, not ours, and it would not survive a keyfunc
		that returned the PEM bytes.
	*/
	if !errors.Is(err, appjwt.ErrUnexpectedSigningMethod) {
		t.Errorf("rejection should come from the RSA algorithm pin, got %v", err)
	}
}

func TestRSAManagerVerifyRejectsAlgNone(t *testing.T) {
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

	if _, err := newRSAManager(t).Verify(token); !errors.Is(err, appjwt.ErrInvalidToken) {
		t.Fatalf("Verify alg:none token: err = %v, want ErrInvalidToken", err)
	}
}

// A token from a different keypair must fail even though both are valid RS256.
func TestRSAManagerVerifyRejectsForeignKey(t *testing.T) {
	token, err := newRSAManager(t).Sign("user-123")
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}

	if _, err := newRSAManager(t).Verify(token); !errors.Is(err, appjwt.ErrInvalidToken) {
		t.Fatalf("Verify token from another keypair: err = %v, want ErrInvalidToken", err)
	}
}

func TestRSAManagerVerifyRejectsExpiredToken(t *testing.T) {
	m := newRSAManager(t)

	token, err := m.SignWithTTL("user-123", -time.Hour)
	if err != nil {
		t.Fatalf("SignWithTTL: %v", err)
	}

	if _, err := m.Verify(token); !errors.Is(err, appjwt.ErrExpiredToken) {
		t.Fatalf("Verify expired token: err = %v, want ErrExpiredToken", err)
	}
}

// The interface is the promise that a feature can move from HS256 to RS256
// without touching its call sites; assert it holds for a value, not just at
// compile time in the package itself.
func TestBothManagersSatisfyTokenManager(t *testing.T) {
	managers := map[string]appjwt.TokenManager{
		"HS256": newHMACManager(t),
		"RS256": newRSAManager(t),
	}

	for name, m := range managers {
		token, err := m.Sign("user-123")
		if err != nil {
			t.Fatalf("%s Sign: %v", name, err)
		}
		claims, err := m.Verify(token)
		if err != nil {
			t.Fatalf("%s Verify: %v", name, err)
		}
		if claims.Subject != "user-123" {
			t.Errorf("%s Subject = %q, want %q", name, claims.Subject, "user-123")
		}
	}
}
