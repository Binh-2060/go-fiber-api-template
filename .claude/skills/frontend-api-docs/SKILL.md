---
name: frontend-api-docs
description: Generate frontend-facing API docs for one feature (or all) of this Fiber API — every endpoint's method + full URL, auth header, path/query params, request body, response body, validation rules, error messages, TypeScript types and UI hints — so a frontend dev or an AI UI generator can build screens without reading Go. Use when asked for "API docs for frontend", "docs for UI", "document the <feature> endpoints", or /frontend-api-docs <feature>.
argument-hint: "<feature> | all   (e.g. users, or a path like examples/crud)"
---

# Frontend API docs

Write Markdown (`.md`) files only, all under `docs/frontend/` (create the folder if missing): `docs/frontend/<feature>.md` (one file per feature) plus `docs/frontend/README.md` (shared rules, written once). Follow `template.md` in this folder exactly.

**Source of truth is the Go code, never guesses.** Every field, rule, status and message in the doc must be traceable to a line you read. If something can't be determined (e.g. a handler returns an untyped `fiber.Map`), write `TODO: confirm with backend` — don't invent it.

## 0. Input

- `$ARGUMENTS` = feature name (`users`) → look in `internal/api/`.
- A path (`examples/crud`, `examples/login`) → document that tree instead.
- `all` → every `Set<Feature>Route` called from `internal/api/routes/routes.go`.
- Nothing given, or the feature isn't registered in `routes.go` → list the features you found and ask which one.

## 1. Collect endpoints (routes)

1. Base URL: `/api/{API_VERSION}` (from `cmd/api/main.go`; `API_VERSION` in `.env`/`.env.development`, e.g. `v1`).
2. Group path: in `internal/api/routes/routes.go`, find `router.Group("/<x>")` passed to `Set<Feature>Route`. For examples, read the group from the route file's doc comment.
3. In `routes/<feature>.go`, record for each `router.<Method>(path, ...handlers)`:
   - method, full URL = base + group + path (path params keep `:id` form),
   - every handler before the controller is middleware. Middleware on a `router.Group(..., mw)` applies to all routes on that group.
4. **Auth:** if a middleware reads `Authorization` (e.g. `RequireAuth`), the endpoint needs `Authorization: Bearer <access_token>`. Read the middleware to copy its exact 401 messages.

## 2. Per endpoint, read the controller

Find the handler in `controllers/<feature>.go` and classify each input:

| Code in controller | Doc section | Notes |
|---|---|---|
| `validators.ParseAndValidateBody(c, &body)` | Request body (JSON) | struct in `schemas/requestbody/`; keys = `json:"..."` tags |
| `validators.ParseAndValidateQueryParam(c, &q)` | Query params | keys = `query:"..."` tags |
| `c.Params("id")` + `validators.ValidateUUID` | Path param, UUID | bad value → 400 `Invalid uuid` |
| `c.Params(...)` without validation | Path param, string | |
| `middlewares.UserID(c)` etc. | nothing from client | comes from the token |

Response, from the `presenters` call:

- `ResponseSuccess(x)` → `items` is `x`. Follow `x` to its type: the service's return type, usually a `schemas/responsebody` struct (keys = `json` tags). `ResponseSuccess("SUCCESS")` → `items: "SUCCESS"`; `ResponseSuccess(nil)` → `items: null`; `fiber.Map{...}` → use its literal keys.
- `ResponseSuccessListData(list, cur, count, total)` → `items: { listData: T[], pagination: { currentPage, currentPageTotalItem, totalPage } }`. If `-1` is passed, say pagination is unused (all `-1`).
- Default paging values (e.g. `defaultPage`, `defaultPerPage`) are in the service; copy them.

## 3. Translate Go into frontend terms

**Types** (for the TypeScript block):

| Go | TS | |
|---|---|---|
| `string` | `string` | |
| `int`, `int64`, `float64` | `number` | |
| `bool` | `boolean` | |
| `time.Time` | `string` | ISO 8601 |
| `*T` | `T \| null` | response: sent as `null`, never omitted |
| `[]T` | `T[]` | never `null`; empty = `[]` |
| request `*T` + `omitempty` | `field?: T \| null` | optional |
| request non-pointer, no `required` | `field?: T` | zero value (`""`, `0`) = "not supplied" |

**Validation tags** (`validate:"..."`) → form rules, stated for humans:

- `required` → required · `omitempty` → optional
- `max=N` / `min=N` → strings: max/min N characters; numbers: max/min value N
- `email`, `uuid`, `url` → format · `oneof=a b c` → select with those options
- `len=N`, `gt`, `gte`, `lt`, `lte`, `eqfield=X` → state literally
- Tag not listed here → copy it verbatim and mark it `(go-playground rule)`.

## 4. Errors, exactly as the server sends them

All errors use the envelope `{ timestamp, status: 0, items: null, error: "<message>" }` (the `ErrorHandler` in `cmd/api/main.go`). Build the endpoint's error table from what the code really returns:

- **400** from `ParseAndValidateBody` / `ParseAndValidateQueryParam`: body is `Field '<GoFieldName>' | Needs to pass '<tag>' validation`. **The field name is the Go struct field (`PerPage`), not the JSON key (`perPage`).** List the Go→JSON mapping so the UI can map it back to a form field. Malformed JSON also gives 400 (Fiber's bind error).
- **400** `Invalid uuid` for validated path params.
- **401** messages from the auth middleware / controller.
- **Feature errors:** every `errors.New("...")` in `exceptions/<feature>.go` the endpoint can reach (trace repository → service → controller). Use the **status the controller actually uses**: by project rule that is usually **500** even for "not found". Write 500, not 404. Add a note that the UI should branch on the `error` string.
- Any `errors.Is(err, X)` branch in the controller with its own status (e.g. login's 401).
- 429 if `internal/config/limiter` is mounted in `main.go`, with its limit/window.

## 5. UI hints (for AI UI generation)

Per feature, derive from the endpoints only:

- **Screens:** list endpoint → table/list page with pagination (+ search input if there's a `q`-style filter); get → detail view; create/update → form; delete → confirm dialog.
- **Form spec:** each body field → input type (`email` → email input, `min=8` password → password input, `oneof` → select, bool → checkbox, `max=255` → maxLength), required marker, client-side rule matching the server rule.
- **Flows:** what to do after success (update returns `"SUCCESS"`, not the object → refetch), on 401 (go to login), on each error message.

## 6. Write and check

1. Write/refresh `docs/frontend/README.md` (shared envelope, base URL, auth, error format — see template) only if missing or out of date.
2. Write `docs/frontend/<feature>.md` from `template.md`. Examples must use real keys and values that pass validation.
3. Self-check before finishing: every endpoint in the route file is documented. Every JSON key matches a tag. No field, status or message comes from a guess. Arrays are shown as `[]`, never `null`.
4. Tell the user the file paths and any `TODO: confirm with backend` items.

Docs only. Don't change Go code, even if you spot a bug. Report it instead.
