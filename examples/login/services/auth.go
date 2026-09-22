package services

import (
	"context"
	"errors"
	"time"

	"github.com/Binh-2060/go-application-template/examples/login/repositories"
	"github.com/Binh-2060/go-application-template/examples/login/schemas/requestbody"
	"github.com/Binh-2060/go-application-template/examples/login/schemas/responsebody"
	appjwt "github.com/Binh-2060/go-application-template/pkg/jwt"
	"golang.org/x/crypto/bcrypt"
)

/*
ErrInvalidCredentials covers both "no such user" and "wrong password".

A caller must not be able to tell the two apart — that's what lets an
attacker enumerate registered emails one login attempt at a time.
*/
var ErrInvalidCredentials = errors.New("services: invalid email or password")

// accessTokenTTL is deliberately short and set here rather than taken from
// JWT_TTL: a login endpoint should hand out short-lived access tokens
// regardless of what other tokens this deployment's default TTL is tuned for.
const accessTokenTTL = 15 * time.Minute

/*
tokenManager is package-level rather than passed into Login on every call,
mirroring pkg/db's Pool(): one instance built once at startup, used by every
request afterwards.

This is the signing half — it needs JWT_RSA_PRIVATE_KEY_PATH. The verifying
half lives in examples/login/middlewares and holds only the public key.
*/
var tokenManager *appjwt.RSAManager

/*
Init builds the RSAManager this package signs with, from JWT_RSA_PRIVATE_KEY_PATH
/ JWT_RSA_PUBLIC_KEY_PATH. Call it once at startup, before any request reaches
Login — see examples/login/README.md for wiring it into cmd/api/main.go.
*/
func Init() error {
	cfg, err := appjwt.RSAConfigFromEnv()
	if err != nil {
		return err
	}

	mgr, err := appjwt.NewRSAManager(cfg)
	if err != nil {
		return err
	}

	tokenManager = mgr
	return nil
}

/*
Login verifies email/password against the in-memory user store and, on
success, signs an access token carrying the user's ID as the JWT subject.

bcrypt.CompareHashAndPassword runs even on a lookup miss to keep this
constant-time between "no such user" and "wrong password" — otherwise the
first branch returning early would let a timing attack distinguish them.
*/
func Login(ctx context.Context, in requestbody.Login) (responsebody.Login, error) {
	user, err := repositories.GetUserByEmail(ctx, in.Email)
	if err != nil && !errors.Is(err, repositories.ErrUserNotFound) {
		return responsebody.Login{}, err
	}

	// A miss above leaves user.PasswordHash empty, which bcrypt always rejects
	// as malformed — so this stays constant-time between "no such user" and
	// "wrong password" without a separate not-found branch.
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(in.Password)); err != nil {
		return responsebody.Login{}, ErrInvalidCredentials
	}

	token, err := tokenManager.SignWithTTL(user.ID, accessTokenTTL)
	if err != nil {
		return responsebody.Login{}, err
	}

	return responsebody.Login{
		AccessToken: token,
		TokenType:   "Bearer",
		ExpiresIn:   int64(accessTokenTTL.Seconds()),
	}, nil
}
