---
name: frontend-api-docs
description: Generate frontend-facing API docs for one feature (or all) of this Fiber API — every endpoint's method + full URL, auth header, path/query params, request body, response body, validation rules, error messages, TypeScript types, Nuxt 4 `$fetch`/`useFetch` usage and UI hints — so a frontend dev or an AI UI generator can build screens without reading Go. Use when asked for "API docs for frontend", "docs for UI", "document the <feature> endpoints", or /frontend-api-docs <feature>.
argument-hint: "<feature> | all   (e.g. users, or a path like examples/crud)"
---

# Frontend API docs

Write Markdown (`.md`) files only, all under `docs/frontend/` (create the folder if missing): `docs/frontend/<feature>.md` (one file per feature) plus `docs/frontend/README.md` (shared rules, written once). Follow `template.md` in this folder exactly.

**Source of truth is the Go code, never guesses.** Every field, rule, status and message in the doc must be traceable to a line you read. If something can't be determined (e.g. a handler returns an untyped `fiber.Map`), write `TODO: confirm with backend` — don't invent it.

**Frontend stack: Nuxt 4 + Nuxt's fetch.** Code examples target Nuxt 4 (`app/` source dir, `shared/types/` for types) and use the wrappers from the README template: `useApi` (a `useFetch` bound to `$api`) for reads, `useNuxtApp().$api` (a `$fetch.create` instance) for mutations. Never raw `fetch()` or axios. Remember that `$fetch`/`useFetch` **throw on status ≥ 400** and put the response body on `error.data`, so error handling is `try/catch` + `apiErrorMessage(err)`, not `if (json.status === 0)`.

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

**Types** (for the TypeScript block, written as `shared/types/<feature>.ts`, all `export`ed so Nuxt auto-imports them). Also export a `<feature>FieldMap` of every Go field name that can appear in a 400 message → its JSON key:

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
- 429 if `internal/config/limiter` is mounted in `main.go`, with its limit/window. Fiber's default limiter replies with plain text `Too Many Requests`, not the envelope, so `error.data` is a string there (the README's `apiErrorMessage` handles it).
- `$fetch` sends a plain-object `body` as JSON with the right `Content-Type`, so the "wrong content type → `Unprocessable Entity`" 400 can't happen from `$api`. Don't list it in the per-endpoint tables.

## 5. UI hints (for AI UI generation)

Per feature, derive from the endpoints only:

- **Pages (Nuxt file routing):** list endpoint → `app/pages/<feature>/index.vue`, table with pagination (+ search input if there's a `q`-style filter); get → `app/pages/<feature>/[id].vue`; create → `new.vue`, update → `[id]/edit.vue`; delete → confirm dialog. Use the backend's resource name for `<feature>`, not its route verbs (`/users/getData` → `app/pages/users/index.vue`).
- **Data:** reads via `useApi` with query params as refs (changing them refetches); mutations via `$api` in handlers.
- **Form spec:** each body field → input type (`email` → email input, `min=8` password → password input, `oneof` → select, bool → checkbox, `max=255` → maxLength), required marker, client-side rule matching the server rule. On 400, map the Go field through `<feature>FieldMap` to the form field.
- **Flows:** what to do after success (update returns `"SUCCESS"`, not the object → `refresh()` or `navigateTo`), on 401 (the `$api` plugin goes to `/login`), on each error message.
- Don't pick a UI library or state store; the frontend owns those.

## 6. Write and check

1. Write/refresh `docs/frontend/README.md` (Nuxt setup, shared envelope, base URL, auth, error helpers — see template) only if missing or out of date.
2. Write `docs/frontend/<feature>.md` from `template.md`. Examples must use real keys and values that pass validation.
3. Self-check before finishing: every endpoint in the route file is documented. Every JSON key matches a tag. No field, status or message comes from a guess. Arrays are shown as `[]`, never `null`. Every code example uses `useApi` / `$api` (no raw `fetch`, no `JSON.stringify` body, no manual `Content-Type`). Paths in `useApi` / `$api` are relative to `apiBase` (`/users/getData`, not `/api/v1/users/getData`).
4. Tell the user the file paths and any `TODO: confirm with backend` items.

Docs only. Don't change Go code, even if you spot a bug. Report it instead.
