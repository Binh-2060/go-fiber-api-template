//go:build integration

package tests

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/Binh-2060/go-application-template/examples/crud/exceptions"
	"github.com/Binh-2060/go-application-template/examples/crud/routes"
	"github.com/Binh-2060/go-application-template/examples/crud/schemas/responsebody"
	"github.com/Binh-2060/go-application-template/internal/api/validators"
	"github.com/gofiber/fiber/v3"
)

/*
End-to-end tests through the Fiber app.

	go test -tags=integration ./examples/crud/...

These cover what the repository tests cannot see: that binding and validation
actually run, that a sentinel becomes the right status code, and that every
response carries the project's envelope. They drive the app with app.Test, so
nothing listens on a port and no test needs a free one.
*/

/*
Build an app with just what these tests exercise: the error handler, the
validators, and this feature's routes.

The ErrorHandler is copied from cmd/api/main.go because main.go builds its app
inline, so there is nothing importable to reuse. That duplication is the reason
these tests can drift from production — extracting a newApp() in main.go and
calling it here would fix it.

CORS, compression and helmet are left out deliberately: they are global concerns
tested where they are configured, and including them here would only obscure
which layer produced a given response.
*/
func newTestApp() *fiber.App {
	app := fiber.New(fiber.Config{
		ErrorHandler: func(ctx fiber.Ctx, err error) error {
			code := fiber.StatusInternalServerError

			var e *fiber.Error
			if errors.As(err, &e) {
				code = e.Code
			}

			return ctx.Status(code).JSON(fiber.Map{
				"timestamp": time.Now().Format("2006-01-02-15-04-05"),
				"status":    0,
				"items":     nil,
				"error":     err.Error(),
			})
		},
	})

	validators.Init()
	routes.SetUserRoute(app.Group("/users"))

	return app
}

// The response shape every endpoint must produce, success or failure.
type envelope struct {
	Timestamp string          `json:"timestamp"`
	Status    int             `json:"status"`
	Items     json.RawMessage `json:"items"`
	Error     *string         `json:"error"`
}

// The extra nesting ResponseSuccessListData adds inside items.
type listItems struct {
	ListData   json.RawMessage `json:"listData"`
	Pagination struct {
		CurrentPage          int `json:"currentPage"`
		CurrentPageTotalItem int `json:"currentPageTotalItem"`
		TotalPage            int `json:"totalPage"`
	} `json:"pagination"`
}

/*
Issue a request and decode the envelope. body may be nil for GET and DELETE.
*/
func do(t *testing.T, app *fiber.App, method, target string, body any) (int, envelope) {
	t.Helper()

	var reader io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal request: %v", err)
		}
		reader = bytes.NewReader(raw)
	}

	req, err := http.NewRequestWithContext(t.Context(), method, target, reader)
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	res, err := app.Test(req, fiber.TestConfig{Timeout: 10 * time.Second, FailOnTimeout: true})
	if err != nil {
		t.Fatalf("%s %s: %v", method, target, err)
	}
	defer res.Body.Close()

	raw, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}

	var env envelope
	if err := json.Unmarshal(raw, &env); err != nil {
		t.Fatalf("%s %s returned non-envelope body %q: %v", method, target, raw, err)
	}

	return res.StatusCode, env
}

func decodeUser(t *testing.T, items json.RawMessage) responsebody.User {
	t.Helper()

	var user responsebody.User
	if err := json.Unmarshal(items, &user); err != nil {
		t.Fatalf("decode user from %q: %v", items, err)
	}

	return user
}

/*
A created resource answers 200 and is immediately readable at its own id — the
one path every other test depends on.
*/
func TestHTTP_CreateThenGet(t *testing.T) {
	app := newTestApp()
	m := marker(t)

	status, env := do(t, app, http.MethodPost, "/users/newData", map[string]string{"name": m + "Ada"})
	if status != http.StatusOK {
		t.Fatalf("POST /users/newData = %d, want 200 (%v)", status, env.Error)
	}
	if env.Status != 1 {
		t.Errorf("envelope status = %d, want 1", env.Status)
	}
	if env.Error != nil {
		t.Errorf("envelope error = %v, want null", *env.Error)
	}

	created := decodeUser(t, env.Items)
	if created.ID == "" {
		t.Fatal("created user has no id")
	}

	status, env = do(t, app, http.MethodGet, "/users/info/"+created.ID, nil)
	if status != http.StatusOK {
		t.Fatalf("GET /users/info/:id = %d, want 200", status)
	}

	got := decodeUser(t, env.Items)
	if got.ID != created.ID || got.Name != m+"Ada" {
		t.Errorf("fetched %+v, want the row just created", got)
	}
}

/*
A malformed id is bad input, not a missing resource: 400, and no query is sent.

The distinction matters — a malformed id could never be valid, so it is the
client's mistake, not a missing row.
*/
func TestHTTP_MalformedIDIs400(t *testing.T) {
	app := newTestApp()

	status, env := do(t, app, http.MethodGet, "/users/info/not-a-uuid", nil)

	if status != http.StatusBadRequest {
		t.Fatalf("GET /users/info/not-a-uuid = %d, want 400", status)
	}
	if env.Status != 0 {
		t.Errorf("envelope status = %d, want 0", env.Status)
	}
	if env.Error == nil {
		t.Error("envelope error is null on a failed request")
	}
}

/*
A well-formed id with no row behind it. The controller returns every service
error as a 500 with err.Error() as the message, so the client sees
"user not found".
*/
func TestHTTP_MissingUserIs500(t *testing.T) {
	app := newTestApp()

	status, env := do(t, app, http.MethodGet, "/users/info/00000000-0000-0000-0000-000000000000", nil)

	if status != http.StatusInternalServerError {
		t.Fatalf("GET missing user = %d, want 500", status)
	}
	if env.Error == nil || *env.Error != exceptions.ErrUserNotFound.Error() {
		t.Errorf("envelope error = %v, want %q", env.Error, exceptions.ErrUserNotFound.Error())
	}
}

/*
Updating an id with no row behind it fails the same way.
*/
func TestHTTP_PutMissingUserIs500(t *testing.T) {
	app := newTestApp()

	status, _ := do(t, app, http.MethodPut, "/users/update/00000000-0000-0000-0000-000000000000",
		map[string]string{"name": marker(t) + "Ada"})

	if status != http.StatusInternalServerError {
		t.Fatalf("PUT missing user = %d, want 500", status)
	}
}

/*
Validation tags are enforced only when input goes through the validators
package — Fiber's own StructValidator hook is not configured on this app. This
fails if a handler is ever rewritten to call c.Bind().Body directly.
*/
func TestHTTP_ValidationRejectsBadBodies(t *testing.T) {
	app := newTestApp()

	cases := map[string]map[string]any{
		"missing name":  {},
		"empty name":    {"name": ""},
		"name over 255": {"name": string(bytes.Repeat([]byte("a"), 256))},
	}

	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			status, _ := do(t, app, http.MethodPost, "/users/newData", body)
			if status != http.StatusBadRequest {
				t.Errorf("POST /users/newData = %d, want 400", status)
			}
		})
	}
}

/*
name is required on PUT too, so an empty body cannot be turned into an
UPDATE. This is the client's mistake: 400, not 500.
*/
func TestHTTP_PutWithNoFieldsIs400(t *testing.T) {
	app := newTestApp()
	m := marker(t)

	_, env := do(t, app, http.MethodPost, "/users/newData", map[string]string{"name": m + "Ada"})
	id := decodeUser(t, env.Items).ID

	status, _ := do(t, app, http.MethodPut, "/users/update/"+id, map[string]any{})

	if status != http.StatusBadRequest {
		t.Fatalf("PUT with empty body = %d, want 400", status)
	}
}

/*
PUT answers 200. The controller reports the literal "SUCCESS" rather than
the updated row, so the fetch-back is what actually proves the write landed.
*/
func TestHTTP_PutUpdatesName(t *testing.T) {
	app := newTestApp()
	m := marker(t)

	_, env := do(t, app, http.MethodPost, "/users/newData", map[string]string{"name": m + "Ada"})
	id := decodeUser(t, env.Items).ID

	status, _ := do(t, app, http.MethodPut, "/users/update/"+id, map[string]string{"name": m + "Grace"})
	if status != http.StatusOK {
		t.Fatalf("PUT = %d, want 200", status)
	}

	_, env = do(t, app, http.MethodGet, "/users/info/"+id, nil)
	updated := decodeUser(t, env.Items)
	if updated.Name != m+"Grace" {
		t.Errorf("Name = %q, want %q", updated.Name, m+"Grace")
	}
}

/*
The list endpoint, including the pagination block ResponseSuccessListData adds.

Filtering by the marker is what makes the counts assertable at all — the table
is shared, so an unfiltered total is whatever else happens to be there.
*/
func TestHTTP_ListFilterAndPagination(t *testing.T) {
	app := newTestApp()
	m := marker(t)

	for _, n := range []string{"A", "B", "C"} {
		do(t, app, http.MethodPost, "/users/newData", map[string]string{"name": m + n})
	}

	status, env := do(t, app, http.MethodGet, "/users/getData?q="+m+"&page=1&perPage=2", nil)
	if status != http.StatusOK {
		t.Fatalf("GET /users/getData = %d, want 200", status)
	}

	var items listItems
	if err := json.Unmarshal(env.Items, &items); err != nil {
		t.Fatalf("decode list items: %v", err)
	}

	var users []responsebody.User
	if err := json.Unmarshal(items.ListData, &users); err != nil {
		t.Fatalf("decode listData: %v", err)
	}

	if len(users) != 2 {
		t.Errorf("page has %d rows, want 2", len(users))
	}
	if items.Pagination.CurrentPage != 1 {
		t.Errorf("currentPage = %d, want 1", items.Pagination.CurrentPage)
	}
	if items.Pagination.CurrentPageTotalItem != 2 {
		t.Errorf("currentPageTotalItem = %d, want 2", items.Pagination.CurrentPageTotalItem)
	}
	// Ceiling of 3/2 — the assertion that catches integer division truncating.
	if items.Pagination.TotalPage != 2 {
		t.Errorf("totalPage = %d, want 2", items.Pagination.TotalPage)
	}
}

/*
No matches must serialise as [] rather than null, so a client can iterate the
result without a nil check. This is asserted on the raw JSON because both decode
into an empty Go slice and the distinction would be lost.
*/
func TestHTTP_ListWithNoMatchesIsEmptyArray(t *testing.T) {
	app := newTestApp()

	_, env := do(t, app, http.MethodGet, "/users/getData?q="+marker(t), nil)

	var items listItems
	if err := json.Unmarshal(env.Items, &items); err != nil {
		t.Fatalf("decode list items: %v", err)
	}

	if got := string(items.ListData); got != "[]" {
		t.Errorf("listData = %s, want []", got)
	}
}

/*
Delete answers 200 once; afterwards the row is gone, so both reading it and
deleting it again fail with 500 ("user not found").
*/
func TestHTTP_Delete(t *testing.T) {
	app := newTestApp()
	m := marker(t)

	_, env := do(t, app, http.MethodPost, "/users/newData", map[string]string{"name": m + "Ada"})
	id := decodeUser(t, env.Items).ID

	if status, _ := do(t, app, http.MethodDelete, "/users/"+id, nil); status != http.StatusOK {
		t.Fatalf("DELETE = %d, want 200", status)
	}

	if status, _ := do(t, app, http.MethodGet, "/users/info/"+id, nil); status != http.StatusInternalServerError {
		t.Fatalf("GET after delete = %d, want 500", status)
	}

	// DeleteUser checks rows affected, so a second delete finds nothing.
	if status, _ := do(t, app, http.MethodDelete, "/users/"+id, nil); status != http.StatusInternalServerError {
		t.Fatalf("second DELETE = %d, want 500", status)
	}
}
