package middlewares

import (
	"errors"
	"strings"

	appjwt "github.com/Binh-2060/go-application-template/pkg/jwt"
	"github.com/gofiber/fiber/v3"
)

/*
verifier is the RS256 manager RequireAuth checks tokens against, built once
by Init. Package-level so RequireAuth can be a plain fiber.Handler used
directly as router.Get("/me", middlewares.RequireAuth, ...) — no factory
call and no manager threaded through every route file.
*/
var verifier *appjwt.RSAManager

/*
Init builds the verify-only RS256 manager RequireAuth uses, from
JWT_RSA_PUBLIC_KEY_PATH. Call it once at startup, before any route using
RequireAuth is mounted — see examples/login/README.md.

Discards the private key even when JWT_RSA_PRIVATE_KEY_PATH is set: verifying
only ever needs the public half, so this process's auth middleware holds no
material that could mint a token. That's the property RS256 buys over HS256,
and it's lost the moment the verifying side keeps the signing key around
because the config happened to carry it.
*/
func Init() error {
	cfg, err := appjwt.RSAConfigFromEnv()
	if err != nil {
		return err
	}
	cfg.PrivateKeyPEM = ""

	mgr, err := appjwt.NewRSAManager(cfg)
	if err != nil {
		return err
	}

	verifier = mgr
	return nil
}

/*
RequireAuth rejects a request unless it carries a valid RS256 bearer token.

On success the token's claim goes into c.Locals for downstream handlers,.
*/
func RequireAuth(c fiber.Ctx) error {
	if verifier == nil {
		// Fail closed. A nil verifier means Init never ran, which is a wiring
		// bug — but a route named RequireAuth must never let a request through
		// because of one.
		return errors.New("middlewares: RequireAuth used before Init()")
	}

	tokenString, ok := cutBearer(c.Get(fiber.HeaderAuthorization))
	if !ok {
		return fiber.NewError(fiber.StatusUnauthorized, "missing bearer token")
	}

	claims, err := verifier.Verify(tokenString)
	if err != nil {
		// Verify's errors stay coarse on purpose (pkg/jwt doc comment): the
		// response must not tell a caller whether the token was expired,
		// forged, or issued by someone else.
		return fiber.NewError(fiber.StatusUnauthorized, "invalid or expired token")
	}

	c.Locals("user", claims)
	return c.Next()
}

/*
UserID returns the subject RequireAuth verified for this request.

The bool is false when RequireAuth did not run — a handler mounted without
it, or mounted above it in the chain. Check it: ignoring it turns a missing
middleware into a 200 with an empty user ID instead of a 401.
*/
func UserID(c fiber.Ctx) (string, bool) {
	user := c.Locals("user")
	id, ok := user.(string)
	return id, ok && id != ""
}

// cutBearer splits "Bearer <token>" into its token. The scheme is matched
// case-insensitively because RFC 6750 defines it that way, and clients that
// send "bearer" are not wrong.
func cutBearer(header string) (string, bool) {
	scheme, token, found := strings.Cut(header, " ")
	if !found || !strings.EqualFold(scheme, "Bearer") || token == "" {
		return "", false
	}
	return token, true
}
