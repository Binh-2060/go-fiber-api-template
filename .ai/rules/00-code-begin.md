# Branch rules

Never commit or push to `main` or `develop`.

Before the first edit of every task, run `git branch --show-current`. If it is `main` or `develop`, don't ask — move off it first:

- Branch for this task exists → `git switch <branch>`
- Otherwise → `git pull && git switch -c <type>/<task-name>` (`feature` / `fix` / `chore`, lowercase-dashed, e.g. `feature/user-login`). Uncommitted edits move with it.

Tell the user which branch you're on. Push only that branch (`git push -u origin <branch>`); changes reach `main` / `develop` via PR only. No force-push or local merge into them.

# Naming rules

Go code is camelCase — never `snake_case` or `ALL_CAPS` identifiers:

- Inside one package → `lowerCamelCase`: `userColumns`, `applyUserFilter`, `perPage`.
- Used by another package → `UpperCamelCase`: `CreateUser`, `UserFilter`, `ErrUserNotFound`.
- Acronyms keep one case: `userID`, `ID`, `DSN()`, `HTTPError` — not `userId`, `Id`, `Dsn()`.
- Constants too: `defaultPerPage`, not `DEFAULT_PER_PAGE`.

Route paths are camelCase too: `/users/newData`, `/users/getData`, `/users/info/:id`.

Not Go identifiers, so they keep their own style — don't rename them:

- Database tables and columns: `snake_case` (`users`, `created_at`).
- JSON and query keys (`json:"..."`, `query:"..."`, `form:"..."`): `snake_case`, matching the columns and existing responses (`per_page`, `current_page`, `list_data`). Renaming one breaks clients.
- File names: lowercase, `_` between words (`user_route_test.go`, `users.gen.go`).
- Env vars: `UPPER_SNAKE_CASE` (`JWT_TTL`, `DB_HOST`).
