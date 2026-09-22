# Example: login with pkg/jwt's RS256 manager

A minimal login feature showing how to use [`pkg/jwt`](../../pkg/jwt)'s `RSAManager` to
issue tokens, without a database — the user store here is an in-memory map, seeded at
package init, standing in for [`examples/crud`](../crud/README.md)'s Postgres-backed
repository. The point of this example is `pkg/jwt`, not another CRUD.

```
routes → middlewares (RequireAuth) → controllers → services → repositories (in-memory)
                                             ↕              ↕
                                          schemas      pkg/jwt.RSAManager
```

## Endpoints

| Method | Path     | Body/Header                        | Success | Failure                          |
| ------ | -------- | ------------------------------------ | ------- | ---------------------------------- |
| `POST` | `/login` | `{email, password}`                  | 200     | 401 invalid credentials            |
| `GET`  | `/me`    | `Authorization: Bearer <token>`      | 200     | 401 missing/invalid/expired token  |

`/me` is behind `middlewares.RequireAuth` — it exists only to prove the middleware
runs and verifies the token before the handler sees the request.

RS256 only. There is no HS256 path here: the signing half (`services`) holds the
private key, the verifying half (`middlewares`) holds only the public key, and that
split is the whole reason to prefer RSA over a shared secret.

Success response:

```json
{
  "timestamp": "...",
  "status": 1,
  "items": {
    "access_token": "eyJ...",
    "token_type": "Bearer",
    "expires_in": 900
  },
  "error": null
}
```

The seeded demo account is `demo@example.com` / `password123`.

## Wiring it up

The example is compiled but not mounted, the same as `examples/crud`. Two things need
to happen in `cmd/api/main.go`, in `internal/api/routes/routes.go`, respectively:

1. **Build both managers once, at startup**, before any request can reach the routes.
   Each reads `pkg/jwt.RSAConfigFromEnv` and fails fast — same rationale as `db.Init`
   in `main.go`: a missing or malformed key should fail at startup, not on the first
   login.

   ```go
   import (
       loginmw "github.com/Binh-2060/go-application-template/examples/login/middlewares"
       loginservices "github.com/Binh-2060/go-application-template/examples/login/services"
   )

   func main() {
       // ... existing db.Init, etc.
       if err := loginservices.Init(); err != nil { // signing: needs the private key
           log.Fatal(err)
       }
       if err := loginmw.Init(); err != nil { // verifying: public key only
           log.Fatal(err)
       }
       // ...
   }
   ```

   Two `Init`s rather than one shared manager is deliberate. `middlewares.Init` drops
   `PrivateKeyPEM` even when `JWT_RSA_PRIVATE_KEY_PATH` is set, so the code path that
   checks tokens holds nothing that could mint one. A service that only validates
   tokens issued elsewhere calls `middlewares.Init()` alone and never sets the private
   key var at all.

2. **Mount the route** in `internal/api/routes/routes.go`:

   ```go
   import loginroutes "github.com/Binh-2060/go-application-template/examples/login/routes"

   func SetRoutes(router fiber.Router) {
       loginRoutes := router.Group("/login")
       loginroutes.SetAuthRoute(loginRoutes)
   }
   ```

## Generating a keypair for local dev

`pkg/jwt` reads PEM files, not inline env vars (see `pkg/jwt`'s package docs for why).
Generate a throwaway 2048-bit pair and point the two env vars at it:

```bash
mkdir -p .keys
openssl genrsa -out .keys/jwt_private.pem 2048
openssl rsa -in .keys/jwt_private.pem -pubout -out .keys/jwt_public.pem
```

Add to `.env` (not `.env.development` — see the root `CLAUDE.md` on which file
`dotenv` actually loads):

```
JWT_RSA_PRIVATE_KEY_PATH=.keys/jwt_private.pem
JWT_RSA_PUBLIC_KEY_PATH=.keys/jwt_public.pem
```

`.keys/` should stay out of git — treat it like `.env`.

## Trying it

```bash
curl -X POST localhost:$PORT/api/$API_VERSION/login \
  -H 'Content-Type: application/json' \
  -d '{"email":"demo@example.com","password":"password123"}'
```

A wrong password or unknown email both return the same 401 body — see
`services.ErrInvalidCredentials` — so the endpoint can't be used to enumerate
registered emails.

Take the `access_token` from that response and call the protected route:

```bash
curl localhost:$PORT/api/$API_VERSION/login/me \
  -H "Authorization: Bearer $TOKEN"
```

Omitting the header, sending garbage, or waiting past `expires_in` seconds all
return 401 from `middlewares.RequireAuth` before `controllers.Me` ever runs.

## What each layer is doing

**`repositories/`** — a hardcoded `map[string]models.User`, seeded once via `init()`
with a bcrypt hash computed at startup rather than committed as a literal. Swap this
for a real lookup (`db.Q(ctx)`, as in `examples/crud/repositories`) when adapting this
example to an actual users table.

**`services/`** — owns the `appjwt.TokenManager` as a package-level variable built by
`Init()`, the same shape as `pkg/db`'s `Pool()`. `Login` depends on the
`TokenManager` interface, not `*appjwt.RSAManager`, so switching to `appjwt.Manager`
(HS256) later touches only `Init`. `accessTokenTTL` is set locally to 15 minutes
rather than taken from `JWT_TTL` — a login endpoint should hand out short-lived
tokens regardless of what other tokens this deployment's default TTL is tuned for.

**`controllers/`** — thin, same shape as `examples/crud/controllers`: validate,
call the service, map `ErrInvalidCredentials` to 401, otherwise 500. Everything else
is returned, not rendered — `main.go`'s `ErrorHandler` turns it into JSON.

**`middlewares/`** — `RequireAuth` is a plain `fiber.Handler`, used directly:

```go
protected := router.Group("/", middlewares.RequireAuth)
protected.Get("/me", controllers.Me)
```

It reads the `Authorization: Bearer <token>` header (scheme matched
case-insensitively, per RFC 6750), verifies against the package's own verify-only
`*appjwt.RSAManager` from `Init()`, and stashes the subject in `c.Locals` on success.
`internal/api/middlewares/` doesn't exist yet in the base template (per the root
`CLAUDE.md`, it's created by the first real feature that needs it) — this example's
copy is the pattern to lift into it.

Three details worth keeping when you copy it:

- **Mount it on a group, not per route.** With a group, "requires auth" is a property
  of the group and a route added next month inherits it. A per-route middleware list
  has to be remembered every time, and forgetting it doesn't fail loudly — it serves
  the route to anyone.
- **The Locals key is an unexported `struct{}` type**, not an exported string. Nothing
  outside the package can overwrite a verified subject with an unverified one, which
  a `c.Locals("userID", ...)` elsewhere in the app otherwise would. v3's own requestid
  middleware uses an unexported key for the same reason.
- **Read it back with `middlewares.UserID(c) (string, bool)` and check the bool.**
  It's false when `RequireAuth` didn't run. Discarding it is how a route mounted
  without the middleware returns 200 with an empty user ID instead of 401 —
  `TestProtectedHandlerMountedWithoutMiddlewareRejects` exists to keep that from
  coming back.

`Verify`'s errors stay coarse on purpose — `pkg/jwt` only distinguishes
`ErrExpiredToken` from everything else (`ErrInvalidToken` covers a bad signature,
a wrong issuer, or a forged `alg`), and `RequireAuth` collapses both into the same
401 message rather than exposing which check failed. A `nil` verifier (`Init` never
ran) returns a 500, not a pass-through: a middleware named `RequireAuth` fails closed.

## Tests

`tests/` is untagged — no database, so `go test ./...` runs it on every commit. The
keypair is generated in `TestMain`, and a second one signs the "valid token, wrong
key" case. The suite mounts the real `SetAuthRoute` rather than assembling its own
handler chain, so it catches the wiring drifting from what production mounts.

```bash
go test ./examples/login/...
```
