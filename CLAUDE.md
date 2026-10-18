# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

Read these before changing behavior or structure:

- `PRODUCT.md` is the source of truth for product behavior. Items marked **Future** are out of scope unless asked for.
- `docs/README.md` is the developer reading order. `docs/architecture.md` has the "where to change things" map, `docs/data-model.md` the ownership and sharing rules, `docs/api.md` the HTTP routes, and `docs/gotchas.md` the traps.
- `.github/copilot-instructions.md` has the project working rules for Copilot. The engineering principles below extend them.

## Commands

Tool versions come from `mise.toml` (Go 1.27, Node 22). Prefix commands with `mise exec --`.

```sh
docker compose up -d db                                      # PostgreSQL (needs .env)
goose -dir backend/db/migrations postgres "$DATABASE_URL" up # apply migrations
go run ./backend/cmd                                         # API on :8080
cd frontend && mise exec -- npm run dev                      # Vite on :5173, proxies /api to :8080
```

Tests and checks:

```sh
mise exec -- go vet ./...
mise exec -- go test ./... -count=1
mise exec -- go test ./backend/internal/timeline -run TestName -count=1   # single Go test
cd frontend && mise exec -- npm run test                                  # vitest (jsdom)
cd frontend && mise exec -- npx vitest run src/lib/timeline.test.ts       # single frontend test file
cd frontend && mise exec -- npm run lint                                  # oxlint
cd frontend && mise exec -- npm run build                                 # tsc -b + vite build
```

The Go tests that use PostgreSQL start one container per test package with Testcontainers, so Docker must be running. Each test gets a fresh copy of a migrated template database (`backend/internal/testutil`); a package that uses it calls `testutil.Run` from `TestMain`.

The pre-commit hook (`.githooks/pre-commit`, enabled with `git config core.hooksPath .githooks`) runs prettier/gofmt on staged files through lint-staged, then all the checks above.

## Architecture

The stack is a React + Vite + TanStack Query frontend, a Go `net/http` API, and PostgreSQL through a pgx pool, with Goose migrations.

### Backend

- `backend/cmd/server.go` registers all routes. `timeline.New` gets a `CurrentUser` function that wraps the auth handler's session lookup. This is how timeline handlers get the authenticated user ID.
- `backend/internal/auth` handles registration, login, and logout with Argon2id passwords. The HTTP-only `calendar_session` cookie holds a raw token, and the database stores only its hash.
- `backend/internal/timeline` holds the HTTP handlers for events, tasks, tags, and sharing. `requireUser()` does the auth check. Request validation lives in `validation.go`, and response helpers map `db` sentinel errors to HTTP status codes. Error bodies are plain text.
- `backend/db` is a single `Store` whose SQL is split into files by feature. It uses handwritten pgx queries, not sqlc (see `docs/decisions.md` for why).

Invariants that span several files:

- Authorization is enforced twice: once in the handler and again in the SQL. Every store query filters by the owner's `user_id`, so a guessed UUID must not reach another user's data.
- `timeline_items` is the parent row and the only place ownership is stored. `events` and `tasks` are subtype rows keyed by `timeline_item_id`. Use one transaction when an operation writes both the parent and a subtype row.
- Shared events are projections, not copies. `event_shares` links a recipient to an event, and the owner is derived through `events -> timeline_items.user_id`. The owner sets per-recipient fields on the share row, and recipients cannot edit anything. Keep source data and share data separate.
- Tags: each user has a `default_tag_id`. Every event has at least one tag and a `main_tag_id` that must be one of its attached tags. Deleting a tag reassigns main tags and defaults (see `backend/db/tags.go`).
- All-day events use an exclusive end at midnight after the last day.

### Migrations

- Never edit an applied migration. Add a new numbered file in `backend/db/migrations`.
- The tests read the migration files from that directory in numeric order and apply them without Goose, so a new file is picked up automatically.

### Frontend (`frontend/src`)

- `CalendarApp.tsx` coordinates the pieces; `App.tsx` is only the signed-in/signed-out gate. Code is colocated by feature: `features/<name>/` holds that feature's components, hooks, pure logic, and tests (for example `features/composer`, `features/calendar`, `features/sharing`). `components/` holds only shared UI and the app shell (`TopBar`, `Sidebar`, `DateTimeField`), `hooks/` only generic hooks, and `lib/` helpers used by several features. Import from the source file; there are no barrel `index.ts` files.
- Server state lives in TanStack Query: query keys are in `lib/queryKeys.ts` and invalidation helpers are in `lib/queryInvalidation.ts`. Each feature's queries and mutations live in its folder. Calendar and week state and the browser time/week preferences live in `features/calendar`.
- `lib/api.ts` is the `api<T>()` JSON wrapper that sends cookie credentials. `types.ts` mirrors the backend JSON. Optional fields match the backend `omitempty` tags.
- The week grid is custom: it is not a calendar library. Item position comes from the number of minutes after local midnight (`lib/calendar.ts`, `features/calendar/timelineView.ts`). Use `date-fns` for date math.
- CSS lives in `styles/` and is imported by `App.css`. Design tokens are in `styles/tokens.css`.

## Engineering principles

### Design

- Prefer the simplest maintainable solution. Follow existing conventions unless there is a concrete reason to change them.
- Use a design pattern only when it solves a real problem here. Prefer composition, clear module boundaries, explicit dependencies, and single-purpose functions.
- Avoid duplicated business logic, premature optimization, speculative generality, and new abstractions or dependencies without a concrete need.
- Keep HTTP handlers, domain logic, persistence, and UI separate. Domain rules live in the backend. The frontend is not an authorization boundary.
- Validate untrusted input at system boundaries.
- Preserve existing behavior unless the task requires changing it. Do not implement unrelated `PRODUCT.md` features.
- Comment non-obvious code only: ownership boundaries, derived projections, and SQL that is hard to follow.

### Research and technology choices

- Check installed versions (`mise.toml`, `go.mod`, `frontend/package.json`) before using an API or pattern that may be newer than them.
- For new libraries, framework features, architectural decisions, or patterns that may have changed recently, verify current recommendations first. Prefer official docs for Go, React, TypeScript, PostgreSQL, and the other tools in use. Patterns.dev is a reference for JavaScript and React patterns, not a checklist.
- When a recommendation is uncertain or has real trade-offs, lay out the options briefly instead of presenting a guess as fact.

### Workflow

- Read the relevant code and its conventions before proposing changes.
- For substantial changes, explain the proposed design and its important trade-offs before editing.
- If a product requirement is ambiguous in a way that changes the implementation, or a request conflicts with `PRODUCT.md`, say so before you proceed.
- Add or update tests for changed behavior and important edge cases. Run targeted tests first, then the relevant tests, type checks, and linters before calling the work done.
- Review the final diff for unnecessary complexity, duplicated logic, regressions, and fit with the architecture.
- Be concise. Make focused changes, and report only changes, test results, and caveats.
- Do not stage or unstage files (even when changing already staged files) unless explicitly asked.
- Do not commit. When finished changes are ready to commit, end with a suggested Conventional Commits message that matches the history (e.g. `feat(events): add approximate start and end times`). The user reviews the changes and commits them.
