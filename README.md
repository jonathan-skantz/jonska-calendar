# Jonska Calendar

A small family calendar. Create events and tasks, tag them, share events with other registered users, and see a week grid with weather and sun times.

## Quick start

You need Docker, [mise](https://mise.jdx.dev), and [Goose](https://github.com/pressly/goose). Full steps, ports, and troubleshooting: [docs/getting-started.md](docs/getting-started.md).

```sh
cd frontend && mise exec -- npm install && cd ..  # once
docker compose up -d db                            # PostgreSQL

# terminal 1: API
cd backend
set -a; . ./.env; set +a                           # export DATABASE_URL for goose
goose -dir db/migrations postgres "$DATABASE_URL" up
mise exec -- go run ./cmd                          # API on :8080

# terminal 2: frontend
cd frontend && mise exec -- npm run dev            # app on http://localhost:5173
```

To run the whole app in Docker instead (no hot reload): `docker compose up -d --build`, then open `http://localhost`.

## Documentation

Start with [docs/README.md](docs/README.md), which gives a reading order.

- [PRODUCT.md](PRODUCT.md): what the app does, and what is planned (**Future**).
- [docs/architecture.md](docs/architecture.md): the parts and how a request moves through them.
- [docs/contributing.md](docs/contributing.md): working rules, commit style, and the pre-commit hook.
- [CLAUDE.md](CLAUDE.md): rules for AI coding agents.
