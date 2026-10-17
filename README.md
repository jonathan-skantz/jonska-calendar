# Run

Requirements: Go 1.27, Node 22, Docker, `mise`, and Goose.

## One-time setup

1. Create `.env` with the PostgreSQL variables used by `compose.yaml` and the
   API connection string:

```dotenv
DATABASE_URL=postgres://<user>:<password>@localhost:5432/<database>?sslmode=disable
```

Optional: to enable quick add (natural-language events), put the Anthropic key
in `.env.local`, not `.env`. `.env` is committed; `.env.local` is git-ignored,
and `.claude/settings.json` stops Claude Code from reading it. Without a key the
quick add field shows "Quick add is not configured".

```dotenv
ANTHROPIC_API_KEY=<key>
# ANTHROPIC_MODEL=claude-haiku-5-5
```

2. Install frontend dependencies once:

```sh
cd frontend && mise exec -- npm install
```

3. Start PostgreSQL and apply the migrations:

```sh
docker compose up -d db
goose -dir backend/db/migrations postgres "$DATABASE_URL" up
```

Do not manually rerun an applied migration or edit an applied migration file.
Add a new migration instead. Goose records applied migrations and safely skips
them on later runs.

## Start the project

Run the API from the repository root:

```sh
go run ./backend/cmd
```

In another shell, run the frontend:

```sh
cd frontend && mise exec -- npm run dev
```

The client runs at `http://localhost:5173` and proxies `/api` to
`http://localhost:8080`.

## Restarts and persistence

- `docker compose up -d db` is safe to repeat after Docker or PostgreSQL stops.
- Restarting the API or frontend only requires rerunning its start command.
- Run migrations again only after adding migration files; pending migrations
  are applied once.
- PostgreSQL data is stored in the named `postgres_data` volume. It survives
  container restarts and `docker compose down`.
- Users, sessions, events, tasks, tags, shares, and overrides survive API,
  frontend, Docker, and PostgreSQL container restarts.
- Sessions remain valid across restarts until they expire or are logged out.

## Stop the project

Stop the API and frontend with `Ctrl+C` in their respective terminals. Stop
PostgreSQL without deleting data with:

```sh
docker compose down
```

Use `docker compose down -v` only for the full data reset below.

## Delete all local data

This deletes the PostgreSQL volume, including all users, sessions, events,
tasks, tags, shares, recipient overrides, and migration history. It does not
delete source files or frontend dependencies.

```sh
docker compose down -v
docker compose up -d db
goose -dir backend/db/migrations postgres "$DATABASE_URL" up
```

Use `docker compose down` without `-v` when you only want to stop the project
without deleting data.

Run the full verification suite from the repository root:

```sh
mise exec -- go vet ./... && mise exec -- go test ./... -count=1 && cd frontend && mise exec -- npm run test && mise exec -- npm run build && mise exec -- npm run lint
```

## Commit checks

Enable the tracked pre-commit hook once per clone:

```sh
git config core.hooksPath .githooks
```

The hook formats staged files, runs Go vet and tests, then runs frontend lint,
tests, and build. It formats only staged files and preserves unstaged changes.
