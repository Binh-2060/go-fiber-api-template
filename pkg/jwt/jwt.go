package jwt

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var (
	ErrEmptySecret  = errors.New("jwt: secret must not be empty")
	ErrWeakSecret   = errors.New("jwt: secret too short")
	ErrInvalidToken = errors.New("jwt: invalid token")
	ErrExpiredToken = errors.New("jwt: token expired")

	/*
		Returned by a keyfunc when the token's "alg" is outside the family the
		manager signs with.

		A sentinel rather than an ad-hoc fmt.Errorf so the pin itself is
		assertable: the library also happens to reject a cross-family token on
		a key-type mismatch, which means a test that only checks for rejection
		still passes when the pin is deleted.
	*/
	ErrUnexpectedSigningMethod = errors.New("jwt: unexpected signing method")
)

/*
RFC 7518 §3.2: an HS256 key must be at least as long as the hash output, i.e.
256 bits. A short secret is brute-forceable offline from a single captured
token, so "dev" is rejected the same way a 1024-bit RSA key is.
*/
const minSecretLen = 32

/*
Tolerance for clock skew between the signing and verifying hosts.

Without it a token minted on a host a couple of seconds ahead is rejected as
not-yet-valid, and one expiring at the boundary is rejected a moment early.
*/
const leeway = 30 * time.Second

/*
TokenManager is the surface a feature should depend on.

Both Manager (HS256) and RSAManager (RS256) implement it, so controllers and
middleware can be written against the interface and the signing scheme can
change without touching call sites.
*/
type TokenManager interface {
	Sign(subject string) (string, error)
	SignWithTTL(subject string, ttl time.Duration) (string, error)
	Verify(tokenString string) (*Claims, error)
}

var (
	_ TokenManager = (*Manager)(nil)
	_ TokenManager = (*RSAManager)(nil)
)

// Claims is the payload carried by every token this package issues.
type Claims struct {
	jwt.RegisteredClaims
}

// newClaims builds the registered-claim set both managers sign, so a change
// to what every token carries lands in one place.
func newClaims(issuer, subject string, ttl time.Duration) Claims {
	now := time.Now()
	return Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    issuer,
			Subject:   subject,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
		},
	}
}

/*
verifyToken parses tokenString with keyFunc and the shared validation options.

Shared by both managers so leeway, issuer checking and error mapping can't
drift between HS256 and RS256. The returned sentinels stay coarse — a caller
must not be able to tell a bad signature from a wrong issuer — but the cause
is wrapped for server-side logs.
*/
func verifyToken(tokenString, issuer string, keyFunc jwt.Keyfunc) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, keyFunc,
		jwt.WithIssuer(issuer),
		jwt.WithLeeway(leeway),
	)

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, fmt.Errorf("%w: %w", ErrExpiredToken, err)
		}
		return nil, fmt.Errorf("%w: %w", ErrInvalidToken, err)
	}

	if !token.Valid {
		return nil, ErrInvalidToken
	}

	return claims, nil
}

// Manager signs and verifies HS256 tokens with a fixed secret and issuer.
type Manager struct {
	secret []byte
	issuer string
	ttl    time.Duration
}

/*
Build a Manager from cfg.

Returns ErrEmptySecret rather than letting a blank JWT_SECRET silently
produce forgeable tokens, and ErrWeakSecret for one short enough to recover
by brute force.
*/
func NewManager(cfg Config) (*Manager, error) {
	if cfg.Secret == "" {
		return nil, ErrEmptySecret
	}
	if len(cfg.Secret) < minSecretLen {
		return nil, fmt.Errorf("%w: need >= %d bytes, got %d", ErrWeakSecret, minSecretLen, len(cfg.Secret))
	}

	return &Manager{
		secret: []byte(cfg.Secret),
		issuer: cfg.Issuer,
		ttl:    cfg.TTL,
	}, nil
}

// Sign issues a token for subject, valid for the Manager's configured TTL.
func (m *Manager) Sign(subject string) (string, error) {
	return m.SignWithTTL(subject, m.ttl)
}

// SignWithTTL issues a token for subject with a caller-chosen lifetime,
// letting the same Manager mint both short-lived access tokens and
// longer-lived refresh tokens.
func (m *Manager) SignWithTTL(subject string, ttl time.Duration) (string, error) {
	claims := newClaims(m.issuer, subject, ttl)

	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(m.secret)
	if err != nil {
		return "", fmt.Errorf("jwt: sign: %w", err)
	}
	return signed, nil
}

/*
Parse and validate tokenString, returning its claims.

Rejects tokens signed with anything other than HMAC (mitigates the
alg:"none" / algorithm-confusion class of attack) and tokens whose issuer
doesn't match the Manager's.
*/
func (m *Manager) Verify(tokenString string) (*Claims, error) {
	return verifyToken(tokenString, m.issuer, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("%w: %v", ErrUnexpectedSigningMethod, t.Header["alg"])
		}
		return m.secret, nil
	})
}
