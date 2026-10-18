# Decisions

Each entry gives what was chosen, why, and when to reconsider. **TODO(maintainer)** marks a reason that still needs writing. No reason has been invented.

## Stack

| Layer          | Choice                                                     | Source                      |
| -------------- | ---------------------------------------------------------- | --------------------------- |
| Backend        | Go 1.27; standard `net/http` with method and path patterns | `mise.toml`, `go.mod`       |
| Database       | PostgreSQL 18 (dev), through the pgx driver                | `compose.yaml`              |
| Migrations     | Goose, SQL files                                           | `backend/db/migrations`     |
| Passwords      | Argon2id                                                   | `backend/internal/auth`     |
| Frontend       | React 19, Vite, TypeScript                                 | `frontend/package.json`     |
| Server state   | TanStack Query                                             | `frontend/package.json`     |
| Dates          | `date-fns`                                                 | `frontend/package.json`     |
| Color contrast | `culori` (WCAG ratio for tag text)                         | `frontend/package.json`     |
| Frontend tests | Vitest, jsdom, Testing Library                             | `frontend/package.json`     |
| Lint           | oxlint                                                     | `frontend/package.json`     |
| Quick add      | Anthropic Messages API, official Go SDK                    | `backend/internal/quickadd` |
| Weather        | Open-Meteo, cached in PostgreSQL                           | `backend/internal/weather`  |
| Tool versions  | mise                                                       | `mise.toml`                 |

**TODO(maintainer):** why Go, PostgreSQL, React, Vite, TanStack Query, and Goose were chosen. The repo does not say.

## The custom week grid

- **Chosen:** a hand-written grid. Items are placed by minutes after local midnight. Hour height is a CSS variable.
- **Why:** it shows things a calendar component does not: approximate edges, commute and gathering bands, heads-up and skipped states, elapsed overlays, sun lines, and night shading.
- **Trade-off:** dragging is intentionally absent, so a calendar library's main benefit does not apply.
- **Reconsider if:** you need drag-and-drop, month or agenda views, or side-by-side overlap columns. See [frontend](frontend.md#the-week-grid).

## pgx, not sqlc

- **Chosen:** hand-written SQL with pgx, split by feature in `backend/db`.
- **Considered:** [sqlc](https://docs.sqlc.dev). It generates typed Go code from SQL and supports pgx/v5 and a UUID override for `google/uuid`.
- **Why not now:**
  - several operations are transactions with validation between queries, which generated methods handle poorly;
  - the SQL is already grouped by feature, and the store is small;
  - adopting sqlc needs query files, config, generated code, and the UUID override.
- **Benefit it would bring:** generated parameter and result types, which removes manual row scanning. It would be a development tool, not a runtime dependency.
- **Reconsider if:** repeated scan code or the query count grows noticeably. The store had 38 exported methods when this was written. Measure before deciding.

## No service layer

- **Chosen:** handlers call the store directly. Rules that touch one store live in store methods. Rules shared across handlers live in `backend/internal/timeline`.
- **Why:** one store, most operations are single transactions. A service layer would add indirection without removing code.
- **Reconsider if:** a workflow spans several stores, or a background job needs the same rules.

## Authorization in two places

- **Chosen:** the handler checks the user, and every store query filters by `user_id`.
- **Why:** if one layer is missed, the other still protects the data. A guessed UUID is not enough.
- **Cost:** the rule is written twice. Keep both in step when ownership logic changes.

## Shares are projections

- **Chosen:** a share is a row that points at the event. The recipient's view is built on read, with the owner's values as the default.
- **Why:** the owner's edits reach recipients automatically. There is no copy to sync and no second source of truth.
- **Trade-off:** each read joins tables. Recipients see only what the owner allows. Per-recipient rules beyond four text fields are not built.

## Cookie sessions, not JWTs

- **Chosen:** a random token in an `HttpOnly`, `SameSite=Lax` cookie. Only its SHA-256 is stored.
- **Why:** deleting a row revokes a session. JavaScript cannot read the token.
- **Trade-off:** no `Secure` flag and no CSRF token. Cross-site protection relies on `SameSite=Lax`.
- **Revisit before:** the app is served outside a trusted local network.
- **TODO(maintainer):** confirm the intended deployment, because `Secure` depends on it.

## Weather cache rounded to 0.1 degrees

- **Chosen:** coordinates are rounded to 0.1 degrees (about 10 km) before caching. Raw Open-Meteo codes are stored, and icons are chosen at read time.
- **Why:** users in one area share one upstream call, and exact positions are never stored. Changing the icon rule needs no refetch.

## Quick add is parse-only

- **Chosen:** `POST /api/events/parse` returns a draft and never writes. The user saves through the normal event API.
- **Why:** the model's output is untrusted. The user reviews every field before it becomes data.

## Reminders are client-driven

- **Chosen:** the browser sets the timers and reports delivery to the server.
- **Why:** there is no push service or background worker.
- **Trade-off:** reminders do not fire when the browser is closed. See [gotchas](gotchas.md#not-built-yet).

## Dependency policy

- Add a dependency only with a concrete need.
- No schema library such as zod. Validation is small, and schemas would not be shared with the Go API.
- No router library. `net/http` patterns are enough.
- Vitest and Testing Library for frontend tests.
- `culori` only for color contrast math.

## Future pressure

[PRODUCT.md](../PRODUCT.md) lists the **Future** items. The design keeps the owner's data separate from the recipient's view, so new features can follow the same pattern:

- uncertain or date-only times;
- recurrence;
- push or background notifications;
- conflict detection between events;
- task scheduling;
- richer sharing rules and tag adoption;
- automation triggers.

New features should add data beside existing data, not change what existing columns mean.
