# Layer rules

Each layer takes and returns only its own types (as in `examples/crud`):

| Layer | Takes | Returns | Imports |
|---|---|---|---|
| Controller | `fiber.Ctx` | `presenters.ResponseSuccess*` / `fiber.NewError` | `services`, `requestbody`, `presenters`, `validators` (+ `middlewares` to read the auth user) |
| Service | `ctx context.Context` first, then a `requestbody` struct or an id | a `responsebody` type (or only `error`) | `repositories`, `models`, `requestbody`, `responsebody`, `pkg/db` |
| Repository | `ctx context.Context` first, then plain values (`name string`, `id string`) or its own filter struct (`UserFilter`) | a `models` type | `models`, `exceptions`, `pkg/db` |

- Controllers never import `repositories` or `models` — they only see what the service returns.
- Services never return a `models` type — convert it with `responsebody.NewX(model)` / `NewXs(models)`, defined next to the response struct.
- Repositories never import `schemas` — the service unpacks the request body into plain values.

```go
// service
func GetUser(ctx context.Context, id string) (responsebody.User, error) {
	user, err := repositories.GetUserByID(ctx, id) // models.User
	if err != nil {
		return responsebody.User{}, err
	}

	return responsebody.NewUser(user), nil
}
```
