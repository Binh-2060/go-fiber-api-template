# Controller response rules

When a controller calls a service and sends its value to the client, name the variables by what kind of response it is — not by the domain (`user`, `token`, `order`):

- **Single value (no pagination)** → `result, err := services.FunctionName(...)`, sent with `presenters.ResponseSuccess(result)`.
- **Paginated list** → `page, err := services.ListX(...)`, unpacked into `presenters.ResponseSuccessListData(page.Users, page.CurrentPage, page.CurrentPageTotalItem, page.TotalPage)` (as in `examples/crud`).
- **No value (update, delete)** → `if err := services.FunctionName(...); err != nil { ... }`, then `presenters.ResponseSuccess("SUCCESS")`.

```go
result, err := services.GetUser(c.Context(), id)
if err != nil {
	return fiber.NewError(fiber.StatusInternalServerError, err.Error())
}

return c.Status(fiber.StatusOK).JSON(presenters.ResponseSuccess(result))
```

Not `user, err := ...`, `data, err := ...` or `res, err := ...` — always `result`. The error is always `err`.

Every success response goes through `presenters.ResponseSuccess` / `ResponseSuccessListData` (pass `-1` for the pagination values when unused). Never hand-roll `fiber.Map` — send a `responsebody` struct.
