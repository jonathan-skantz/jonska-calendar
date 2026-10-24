# Decisions

Each entry gives what was chosen, why, and when to reconsider. **TODO(maintainer)** marks a reason that still needs writing. No reason has been invented.

## Stack

| Layer          | Choice                                                     | Source                        |
| -------------- | ---------------------------------------------------------- | ----------------------------- |
| Backend        | Go 1.27; standard `net/http` with method and path patterns | `mise.toml`, `backend/go.mod` |
| Database       | PostgreSQL 18 (dev), through the pgx driver                | `compose.yaml`                |
| Migrations     | Goose, SQL files                                           | `backend/db/migrations`       |
| Passwords      | Argon2id                                                   | `backend/internal/auth`       |
| Frontend       | React 19, Vite, TypeScript                                 | `frontend/package.json`       |
| Server state   | TanStack Query                                             | `frontend/package.json`       |
| Dates          | `date-fns`                                                 | `frontend/package.json`       |
| Color contrast | `culori` (WCAG ratio for tag text)                         | `frontend/package.json`       |
| Frontend tests | Vitest, jsdom, Testing Library                             | `frontend/package.json`       |
| Lint           | oxlint                                                     | `frontend/package.json`       |
| Quick add      | Anthropic Messages API, official Go SDK                    | `backend/internal/quickadd`   |
| Assistant      | Anthropic Messages API, Go SDK tool runner (beta)          | `backend/internal/assistant`  |
| Weather        | Open-Meteo, cached in PostgreSQL                           | `backend/internal/weather`    |
| Tool versions  | mise                                                       | `mise.toml`                   |

**TODO(maintainer):** why Go, PostgreSQL, React, Vite, TanStack Query, and Goose were chosen. The repo does not say.

## The custom week grid

- **Chosen:** a hand-written grid. Items are placed by minutes after local midnight. Hour height is a CSS variable.
- **Why:** it shows things a calendar component does not: approximate edges, commute and gathering bands, heads-up and skipped states, elapsed overlays, sun lines, and night shading.
- **Trade-off:** dragging to move is a small Pointer Events hook (`useEntryDrag`) rather than a library. It only moves items within the visible days, with no resizing; a calendar library's drag features would mostly go unused.
- **Reconsider if:** you need resizing, dragging between weeks, or month or agenda views. See [frontend](frontend.md#the-week-grid).

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

- **Chosen:** handlers call the store directly. Rules that touch one store live in store methods. Rules shared across handlers live in `backend/internal/timeline`, such as `ValidateEvent`, which the assistant uses too.
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

## Subscribed calendars are read-only copies, replaced on refresh

- **Chosen:** a subscription's events live in their own tables (`subscription_events`), are replaced as a whole on each refresh, and show read-only like shared events.
- **Considered:** importing them into the user's own events and updating them by UID on each refresh. That would give them tags and reminders, but every write path (edit, drag, delete, tags, shares, the assistant) would need a read-only check, and a refresh would overwrite local edits.
- **Trade-off:** no tags or reminders on subscribed events, and their IDs change on each refresh.
- **Reconsider if:** reminders on subscribed events are wanted. They would need a stable key, such as the feed's UID.

## Subscriptions refresh in the API process

- **Chosen:** a goroutine started in `server.go` refreshes stale subscriptions, plus a "Refresh now" route. It is the only background job.
- **Why:** the app runs as one process for a family. A separate worker or a queue would add deployment for no gain.
- **Considered:** refreshing on demand when the timeline is loaded. That needs no job, but the week shows old events until the next load, and a slow feed delays the request.
- **Reconsider if:** the API runs as several processes, which would each refresh the same subscriptions.

## Cookie sessions, not JWTs

- **Chosen:** a random token in an `HttpOnly`, `SameSite=Lax` cookie. Only its SHA-256 is stored.
- **Why:** deleting a row revokes a session. JavaScript cannot read the token.
- **Secure flag:** set when `SESSION_COOKIE_SECURE=true`. A public deployment serves HTTPS and sets it. Local development runs on plain HTTP and leaves it off.
- **Trade-off:** no CSRF token. Cross-site protection relies on `SameSite=Lax`.

## Weather cache rounded to 0.1 degrees

- **Chosen:** coordinates are rounded to 0.1 degrees (about 10 km) before caching. Raw Open-Meteo codes are stored, and icons are chosen at read time.
- **Why:** users in one area share one upstream call, and exact positions are never stored. Changing the icon rule needs no refetch.

## Quick add is parse-only

- **Chosen:** `POST /api/events/parse` returns a draft and never writes. The user saves through the normal event API.
- **Why:** the model's output is untrusted. The user reviews every field before it becomes data.

## The assistant is a tool loop in the Go API

- **Chosen:** `backend/internal/assistant` runs the agent with the Go SDK's tool runner (`client.Beta.Messages.NewToolRunner`). The model picks tools (`list_events`, `find_free_slots`, `create_event`, `move_event`, `skip_event`, `delete_event`), sees their results, and repeats, up to 8 model calls per message. The tools call `db.Store` with the signed-in user's ID.
- **Why not the Claude Agent SDK or `claude -p`:** both run Claude Code, a coding agent with its own file and shell tools, as a separate process. They are made for Python or TypeScript programs and for working in a repository, not for a multi-user API. Using them would need a second service, a sandbox, and a way to pass the user's identity. The tool runner is part of the SDK the API already uses, and authorization stays the same as every other route.
- **Writes by risk:** creating one of the user's own events runs at once and can be undone. Moving, skipping and deleting change what exists, so they are queued as actions the user applies or declines. The agent's input includes text from other users (titles of shared events), so a prompt injection can at worst create an event the user sees and can undo.
- **One action model:** every write is an `Action` with a kind and a status, decided through one route. A queued action stores the intent (new times, skipped), not a copy of the event, and is applied to the event as it is then through `timeline.ValidateEvent`, the same rules as the event routes. A new kind of change (sharing) is a new kind plus a case in `change` or `carryOut`.
- **Conversations live in memory** on the server, keyed by ID and owner, for 30 minutes. The client never sends history back, so it cannot forge tool results, and can only decide actions the agent actually created. A restart ends every conversation and drops its undecided actions.
- **Busy rules are written twice:** in Go for the assistant (`busy.go`) and in TypeScript for the week view (`lib/timeline.ts`). Both test suites run `frontend/src/lib/testdata/occupied-spans.json`, so a change to one that is not made to the other fails a test.
- **Reconsider if:** the app runs on more than one server (conversations, actions and rate limits would need the database), or actions should survive a restart.

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
