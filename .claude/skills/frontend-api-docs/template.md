# Templates

Replace `<...>`. Drop a section only when it doesn't apply (e.g. no body on GET).

Frontend stack: **Nuxt 4** (`app/` source dir, `shared/` for code used by app and server) with Nuxt's built-in fetch: `$fetch` (ofetch) and `useFetch`. Every example goes through the `$api` / `useApi` wrappers defined in the README. Never use raw `fetch()` or axios.

---

## `docs/frontend/README.md` (shared — write once)

````markdown
# API for frontend (Nuxt 4)

Base URL: `{HOST}/api/{API_VERSION}` — e.g. `http://localhost:8000/api/v1`.
Health check: `GET /healthz` (no prefix, not in the envelope) → `{ "status": "OK" }`.

## Setup

### `nuxt.config.ts`

```ts
export default defineNuxtConfig({
  runtimeConfig: {
    public: {
      apiBase: "http://localhost:8000/api/v1", // override with NUXT_PUBLIC_API_BASE
    },
  },
});
```

### `app/plugins/api.ts`: one `$fetch` instance for the whole app

```ts
export default defineNuxtPlugin((nuxtApp) => {
  const config = useRuntimeConfig();

  const api = $fetch.create({
    baseURL: config.public.apiBase,
    onRequest({ options }) {
      // <Only if the API has auth — otherwise delete this hook.>
      const token = useCookie("access_token"); // cookie name is the frontend's choice
      if (token.value) options.headers.set("Authorization", `Bearer ${token.value}`);
    },
    async onResponseError({ response }) {
      if (response.status === 401) {
        await nuxtApp.runWithContext(() => navigateTo("/login"));
      }
    },
  });

  return { provide: { api } };
});
```

### `app/composables/useApi.ts`: `useFetch` that uses `$api`

```ts
import type { UseFetchOptions } from "nuxt/app";

export function useApi<T>(url: string | (() => string), options?: UseFetchOptions<T>) {
  return useFetch(url, { ...options, $fetch: useNuxtApp().$api as typeof $fetch });
}
```

Use `useApi` for reads in pages and components (SSR-safe, reactive `query`, gives `refresh()`). Use `useNuxtApp().$api` for mutations in event handlers.

## Response envelope: `shared/types/api.ts`

Every response, success or error:

```ts
export interface ApiResponse<T> {
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

<If auth middleware exists:> Protected endpoints need `Authorization: Bearer <access_token>` (added by the `$api` plugin). A 401 means the token is missing, invalid or expired; the plugin sends the user to `/login`.
<If not:> No endpoint needs a token yet. Delete the `onRequest` hook until auth is added.

## Errors

`$fetch` / `useFetch` **throw on any HTTP status ≥ 400**; the envelope is on `error.data`. Read the message with one helper.

`app/utils/apiError.ts`:

```ts
/** Message to show for an error from $api or useApi. */
export function apiErrorMessage(err: unknown): string {
  const data = (err as { data?: unknown } | null)?.data;
  if (data && typeof data === "object" && "error" in data && typeof data.error === "string") {
    return data.error;                     // normal envelope
  }
  if (typeof data === "string" && data) return data; // plain-text body (e.g. 429)
  return (err as Error | null)?.message ?? "Unexpected error";
}

/** Go field name from a 400 validation message, e.g. "PerPage". */
export function apiErrorField(message: string): string | null {
  return /^Field '(\w+)'/.exec(message)?.[1] ?? null;
}
```

- Validation (400): `Field '<GoFieldName>' | Needs to pass '<rule>' validation`. Only the first failing field is reported. The name is the backend's **Go** name; each feature doc has a `<feature>FieldMap` to turn it into the JSON key / form field.
- Business errors currently come back as **500** with a readable message (e.g. `"user not found"`). Branch on the message, not the status.
- <Rate limit, if mounted: N requests per window per IP → 429 with plain-text body `Too Many Requests`, not the envelope.>

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

## Types: `shared/types/<feature>.ts`

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

// Go field name in 400 messages → JSON key / form field
export const <feature>FieldMap: Record<string, string> = {
  <GoField>: "<json_key>",
};
```

## <METHOD> `<full url>`

<Purpose.> Auth: <none | Bearer>.

**Path params**

| Name | Type | Rule |
|---|---|---|

**Query params**

| Key | Type | Required | Rule | Default | Go field (in 400 errors) |
|---|---|---|---|---|---|

**Body** (`application/json`)

| Key | Type | Required | Rule | Go field (in 400 errors) |
|---|---|---|---|---|

**Nuxt usage**

<GET: read with useApi in a page/component.>

```ts
const page = ref(1);
const { data, status, error, refresh } = await useApi<ApiResponse<<Type>>>("/<group>/<path>", {
  query: { page },           // refs are reactive: changing page refetches
});
const rows = computed(() => data.value?.items?.list_data ?? []);
```

<POST/PUT/DELETE: call $api from an event handler.>

```ts
const { $api } = useNuxtApp();
try {
  const res = await $api<ApiResponse<<Type>>>("/<group>/<path>", {
    method: "<METHOD>",
    body: { <valid example> }, // plain object: $fetch sends it as JSON
  });
  // use res.items
} catch (err) {
  const message = apiErrorMessage(err);
  // map apiErrorField(message) through <feature>FieldMap to highlight a field, else toast message
}
```

**Success — 200**

```json
{ "timestamp": "2026-01-02-15-04-05", "status": 1, "items": <example>, "error": null }
```

**Errors** (thrown; read with `apiErrorMessage(err)`)

| HTTP | `error` message | When | UI action |
|---|---|---|---|
| 400 | `Field 'Name' \| Needs to pass 'required' validation` | name empty | highlight field |
| 500 | `<exception message>` | <when> | <toast / redirect / empty state> |

## UI hints (Nuxt 4)

- **Pages:** <`app/pages/<feature>/index.vue` list with pagination + search · `app/pages/<feature>/[id].vue` detail · `new.vue` / `[id]/edit.vue` forms · delete confirm dialog>
- **State:** <page/per_page/q as refs, ideally synced to `route.query`; reset page to 1 when the search changes; debounce search>
- **Form <Name>:** `<json_key>` → <input type>, <required>, <maxLength/pattern> (client rules mirror the server rules above)
- **After success:** <`refresh()` the list / `navigateTo(...)` / use returned object>
- **Notes:** <gotchas: update returns "SUCCESS" not the object; not-found is 500; etc.>
````
