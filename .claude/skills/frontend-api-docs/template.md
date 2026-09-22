# Templates

Replace `<...>`. Drop a section only when it doesn't apply (e.g. no body on GET).

---

## `docs/frontend/README.md` (shared — write once)

````markdown
# API for frontend

Base URL: `{HOST}/api/{API_VERSION}` — e.g. `http://localhost:8000/api/v1`.
Health check: `GET /healthz` (no prefix).

## Response envelope

Every response, success or error:

```ts
interface ApiResponse<T> {
  timestamp: string;      // "2006-01-02-15-04-05" layout, server local time
  status: 1 | 0;          // 1 = success, 0 = error
  items: T | null;        // payload; null on error
  error: string | null;   // message on error; null on success
}

interface Paginated<T> {
  listData: T[];          // never null; empty = []
  pagination: {
    currentPage: number;
    currentPageTotalItem: number;
    totalPage: number;    // all three are -1 when the endpoint doesn't paginate
  };
}
```

## Auth

Protected endpoints need `Authorization: Bearer <access_token>`. A 401 means the token is missing, invalid or expired. Send the user to login.

## Errors

- Check `status === 0` (or HTTP status ≥ 400), then show `error`.
- Validation (400): `Field '<GoFieldName>' | Needs to pass '<rule>' validation`. The field name is the backend's Go name. Each feature doc maps it to the JSON key.
- Business errors currently come back as **500** with a readable message (e.g. `"user not found"`). Branch on the message, not the status.

## Features

- [<feature>](<feature>.md)
````

---

## `docs/frontend/<feature>.md`

````markdown
# <Feature> API

<One sentence: what this resource is.> Source: `<routes file path>`.

| Method | URL | Auth | Purpose |
|---|---|---|---|
| POST | `/api/v1/<group>/<path>` | — / Bearer | <purpose> |

## Types

```ts
// Response shapes (schemas/responsebody)
export interface <Name> {
  <json_key>: <ts type>;   // <note: nullable / format>
}

// Request shapes (schemas/requestbody)
export interface <Name>Request {
  <json_key>: <ts type>;   // required, max 255 chars
}

export interface <List>Query {
  page?: number;           // min 1, default <n>
}
```

## <METHOD> `<full url>`

<Purpose.> Auth: <none | Bearer>.

**Path params**

| Name | Type | Rule |
|---|---|---|

**Query params**

| Key | Type | Required | Rule | Default |
|---|---|---|---|---|

**Body** (`application/json`)

| Key | Type | Required | Rule | Go field (in 400 errors) |
|---|---|---|---|---|

**Request example**

```ts
const res = await fetch(`${BASE_URL}/<group>/<path>`, {
  method: "<METHOD>",
  headers: { "Content-Type": "application/json" /*, Authorization: `Bearer ${token}` */ },
  body: JSON.stringify({ <valid example> }),
});
const json: ApiResponse<<Type>> = await res.json();
```

**Success — 200**

```json
{ "timestamp": "2026-01-02-15-04-05", "status": 1, "items": <example>, "error": null }
```

**Errors**

| HTTP | `error` message | When | UI action |
|---|---|---|---|
| 400 | `Field 'Name' \| Needs to pass 'required' validation` | name empty | highlight field |
| 500 | `<exception message>` | <when> | <toast / redirect / empty state> |

## UI hints

- **Screens:** <list page w/ pagination + search, detail, create/edit form, delete confirm>
- **Form <Name>:** `<json_key>` → <input type>, <required>, <maxLength/pattern>
- **After success:** <refetch list / navigate / use returned object>
- **Notes:** <gotchas: update returns "SUCCESS" not the object; not-found is 500; etc.>
````
