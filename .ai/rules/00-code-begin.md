# Branch rules

Never commit or push to `main` or `develop`.

Before the first edit of every task, run `git branch --show-current`. If it is `main` or `develop`, don't ask — move off it first:

- Branch for this task exists → `git switch <branch>`
- Otherwise → `git pull && git switch -c <type>/<task-name>` (`feature` / `fix` / `chore`, lowercase-dashed, e.g. `feature/user-login`). Uncommitted edits move with it.

Tell the user which branch you're on. Push only that branch (`git push -u origin <branch>`); changes reach `main` / `develop` via PR only. No force-push or local merge into them.
