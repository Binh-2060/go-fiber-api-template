# API for frontend

Base URL: `{HOST}/api/{API_VERSION}`, e.g. `http://localhost:8000/api/v1` (`PORT=8000`, `API_VERSION=v1` in `.env.development`).
Health check: `GET /healthz` (no prefix) → `{ "status": "OK" }` (this one does not use the envelope).

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
  list_data: T[];         // never null; empty = []
  pagination: {
    current_page: number;
    current_page_total_item: number;
    total_page: number;   // all three are -1 when the endpoint doesn't paginate
  };
}
```

## Auth

No endpoint currently needs a token (`internal/api/middlewares/auth.go` is still empty). When auth is added, protected endpoints will need `Authorization: Bearer <access_token>`, and a 401 will mean the token is missing, invalid or expired. Send the user to login when that happens.

## Errors

- Check `status === 0` (or HTTP status ≥ 400), then show `error`.
- Validation (400): `Field '<GoFieldName>' | Needs to pass '<rule>' validation`. The field name is the backend's Go name, and only the first failing field is reported. Each feature doc maps it to the JSON key.
- A body sent without `Content-Type: application/json` → 400 `Unprocessable Entity`. Malformed JSON → 400 with the parser's message.
- Business errors currently come back as **500** with a readable message (e.g. `"user not found"`). Branch on the message, not the status.
- Rate limit: 1000 requests per 60 s sliding window per client IP. Over the limit → **429** with plain-text body `Too Many Requests` (**not** the JSON envelope).

## Features

- [users](users.md)
