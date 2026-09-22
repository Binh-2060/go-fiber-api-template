package jwt

import (
	"crypto/rsa"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var (
	ErrMissingPublicKey  = errors.New("jwt: public key required")
	ErrMissingPrivateKey = errors.New("jwt: private key required to sign")
)

/*
RSAManager signs and verifies RS256 tokens with an RSA keypair.

Unlike Manager (HS256), signing and verifying don't share a secret: a
service that only validates tokens can be handed just the public key, so a
leak there can't be used to mint new tokens.

Holds exactly one keypair and writes no "kid" header, so rotating keys means
a window where old tokens fail to verify. Add kid selection here before the
first rotation, not during it.
*/
type RSAManager struct {
	privateKey *rsa.PrivateKey // nil for a verify-only Manager
	publicKey  *rsa.PublicKey
	issuer     string
	ttl        time.Duration
}

/*
Build an RSAManager from cfg.

cfg.PublicKeyPEM is always required. cfg.PrivateKeyPEM is only required to
sign — leave it empty to build a verify-only Manager for a service that
checks tokens but never issues them.

Rejects RSA keys smaller than 2048 bits: 1024-bit RSA has been factorable
with realistic compute for years and NIST deprecated it in 2013, so it
shouldn't be used for new tokens.
*/
func NewRSAManager(cfg RSAConfig) (*RSAManager, error) {
	if cfg.PublicKeyPEM == "" {
		return nil, ErrMissingPublicKey
	}

	pub, err := jwt.ParseRSAPublicKeyFromPEM([]byte(cfg.PublicKeyPEM))
	if err != nil {
		return nil, fmt.Errorf("jwt: parse public key: %w", err)
	}
	if bits := pub.N.BitLen(); bits < 2048 {
		return nil, fmt.Errorf("jwt: RSA key too small (%d bits, need >= 2048)", bits)
	}

	m := &RSAManager{publicKey: pub, issuer: cfg.Issuer, ttl: cfg.TTL}

	if cfg.PrivateKeyPEM != "" {
		priv, err := jwt.ParseRSAPrivateKeyFromPEM([]byte(cfg.PrivateKeyPEM))
		if err != nil {
			return nil, fmt.Errorf("jwt: parse private key: %w", err)
		}
		m.privateKey = priv
	}

	return m, nil
}

// Sign issues a token for subject, valid for the Manager's configured TTL.
func (m *RSAManager) Sign(subject string) (string, error) {
	return m.SignWithTTL(subject, m.ttl)
}

// SignWithTTL issues a token for subject with a caller-chosen lifetime.
// Returns ErrMissingPrivateKey on a verify-only Manager.
func (m *RSAManager) SignWithTTL(subject string, ttl time.Duration) (string, error) {
	if m.privateKey == nil {
		return "", ErrMissingPrivateKey
	}

	claims := newClaims(m.issuer, subject, ttl)

	signed, err := jwt.NewWithClaims(jwt.SigningMethodRS256, claims).SignedString(m.privateKey)
	if err != nil {
		return "", fmt.Errorf("jwt: sign: %w", err)
	}
	return signed, nil
}

/*
Parse and validate tokenString, returning its claims.

Rejects tokens signed with anything other than RSA (mitigates the
alg:"none" / algorithm-confusion class of attack, including a switch to
HS256 that would let an attacker sign with the public key as if it were an
HMAC secret) and tokens whose issuer doesn't match the Manager's.
*/
func (m *RSAManager) Verify(tokenString string) (*Claims, error) {
	return verifyToken(tokenString, m.issuer, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("%w: %v", ErrUnexpectedSigningMethod, t.Header["alg"])
		}
		return m.publicKey, nil
	})
}
