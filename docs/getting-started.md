# Getting started

## Prerequisites

| Tool                         | Why                                                                  | Check            |
| ---------------------------- | -------------------------------------------------------------------- | ---------------- |
| [mise](https://mise.jdx.dev) | Installs the Go and Node versions in `mise.toml` (Go 1.27, Node 22). | `mise --version` |
| Docker                       | Runs PostgreSQL, and the Go tests that need a database.              | `docker info`    |
| Goose                        | Applies migrations. Not in `mise.toml`, so install it yourself.      | `goose -version` |

Install Goose:

```sh
go install github.com/pressly/goose/v3/cmd/goose@latest
```

Prefix Go and Node commands with `mise exec --`. Without it, your shell may use the wrong versions.

## Ports

| Port | What                                             | Set in                  |
| ---- | ------------------------------------------------ | ----------------------- |
| 5432 | PostgreSQL (Docker publishes it to your machine) | `compose.yaml`          |
| 8080 | Go API (hard-coded)                              | `backend/cmd/server.go` |
| 5173 | Vite dev server (the page you open)              | Vite default            |

If PostgreSQL already runs on port 5432, stop it first.

## Environment files

| File                 | Committed?       | Contains                                            |
| -------------------- | ---------------- | --------------------------------------------------- |
| `.env`               | Yes              | PostgreSQL settings for `docker compose`. Dev only. |
| `backend/.env`       | Yes              | The API's `DATABASE_URL`. Dev only.                 |
| `backend/.env.local` | No (git-ignored) | API secrets, such as `ANTHROPIC_API_KEY`.           |

`.env` has these keys:

```dotenv
POSTGRES_DB=...
POSTGRES_USER=...
POSTGRES_PASSWORD=...
```

`backend/.env` points at that database:

```dotenv
DATABASE_URL=postgres://<user>:<password>@localhost:5432/<database>?sslmode=disable
```

Who reads them:

- `docker compose` reads `.env` for the `POSTGRES_*` values.
- The API reads `.env`, then `.env.local`, from its working directory (`backend/cmd/server.go`). Run it from `backend/`.
- Goose does **not** read `DATABASE_URL`. Export `backend/.env` into your shell first:

  ```sh
  cd backend && set -a; . ./.env; set +a
  ```

Quick add (the box that turns "lunch with Sam tomorrow noon" into an event) and the assistant need a key in `backend/.env.local`:

```dotenv
ANTHROPIC_API_KEY=<key>
# ANTHROPIC_MODEL=claude-haiku-5-5     # quick add
# ASSISTANT_MODEL=claude-opus-5-5      # assistant
```

Without a key, both show that they are not configured. Everything else still works.

To subscribe to a calendar on your own network, or to a feed of this app on `localhost`, allow private addresses:

```dotenv
SUBSCRIPTIONS_ALLOW_PRIVATE=true
```

See [gotchas](gotchas.md#subscriptions-cannot-reach-private-addresses).

`.claude/settings.json` stops Claude Code from reading `.env.local`.

## First run

From the repository root:

```sh
# 1. Install frontend dependencies (once)
cd frontend && mise exec -- npm install && cd ..

# 2. Start PostgreSQL
docker compose up -d db

# 3. Apply migrations (terminal 1, after exporting backend/.env, see above)
cd backend
goose -dir db/migrations postgres "$DATABASE_URL" up

# 4. Start the API (terminal 1, from backend/)
mise exec -- go run ./cmd

# 5. Start the frontend (terminal 2)
cd frontend && mise exec -- npm run dev
```

Open `http://localhost:5173`, register, and create an event.

The Vite dev server forwards every `/api` request to `http://localhost:8080`. The browser never talks to PostgreSQL directly.

## Day to day

- Restart the API or frontend: run its start command again.
- Stop the database and keep data: `docker compose down`.
- After pulling new migrations: run the goose command again. It applies only files it has not seen.
- Never edit an applied migration. Add a new one ([contributing](contributing.md#migrations)).

## Reset local data

This deletes users, events, tags, shares, and migration history. Source files and `node_modules` are untouched.

```sh
docker compose down -v
docker compose up -d db
cd backend && goose -dir db/migrations postgres "$DATABASE_URL" up
```

## Troubleshooting

| Symptom                                           | Likely cause                                                                          | Fix                                                                    |
| ------------------------------------------------- | ------------------------------------------------------------------------------------- | ---------------------------------------------------------------------- |
| `goose` has an empty `DATABASE_URL` or is refused | `backend/.env` was not exported into the shell.                                       | In `backend/`, run `set -a; . ./.env; set +a`, then retry.             |
| API cannot connect to the database                | The database container is not running, or `DATABASE_URL` does not match `POSTGRES_*`. | `docker compose ps`, then compare `backend/.env` with `.env`.          |
| `docker compose up` fails on port 5432            | Another PostgreSQL is running.                                                        | Stop it, or change the host port in `compose.yaml` and `backend/.env`. |
| Database container exits after an upgrade         | The volume came from an older PostgreSQL image (this repo uses `postgres:18`).        | `docker compose down -v`, then start again. This deletes local data.   |
| API starts but `DATABASE_URL` is empty            | Not run from `backend/`, so `.env` was not found.                                     | Run `go run ./cmd` from `backend/`.                                    |
| Quick add says "not configured"                   | No `ANTHROPIC_API_KEY` in `backend/.env.local`.                                       | Add the key and restart the API. Expected without a key.               |
| Go tests cannot start a container                 | Docker is not running. The first run also downloads an image.                         | Start Docker and retry.                                                |
| `git commit` skips checks                         | The pre-commit hook is not enabled.                                                   | `git config core.hooksPath .githooks`                                  |
