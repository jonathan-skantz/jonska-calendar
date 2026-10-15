# Technical Architecture

This document describes the current demo architecture and the intended path toward the product in `PRODUCT.md`.

## Scope

The demo is a small family calendar. It currently supports:

- email/password registration, login, logout, and cookie sessions;
- owned events with exact start/end times;
- deadline-only tasks and completion state;
- a custom seven-day, 24-hour React week grid;
- personal tags;
- direct event sharing between registered users;
- recipient-local event presentation overrides and private custom info.

The product intentionally does not implement uncertain timing, recurrence, reminders, conflict detection, automatic task scheduling, external calendars, advanced sharing policies, triggers, templates, or backup/import.

## System Shape

```text
React/Vite frontend
        |
        | HTTP JSON + cookie session
        v
Go net/http API
        |
        v
PostgreSQL via pgx pool
        ^
        |
Goose migrations
```

The frontend is a separate development server. Vite proxies `/api` to the Go server at `localhost:8080`. The Go server owns authentication, authorization, validation, and persistence. The browser never connects to PostgreSQL directly.

## Repository Map

### Backend

- [backend/cmd/server.go](../backend/cmd/server.go)
  - Starts the PostgreSQL pool and HTTP server.
  - Constructs auth and timeline handlers.
  - Registers all API routes.
- [backend/internal/auth/handler.go](../backend/internal/auth/handler.go)
  - Registration, login, logout, current-user endpoint, and session-cookie parsing.
  - Password hashing uses Argon2id.
- [backend/internal/timeline/](../backend/internal/timeline/)
  - `handler.go` contains shared handler setup and timeline operations. `requests.go` defines API payloads and responses. Feature files contain event, task, tag, and sharing endpoints. `validation.go` contains request validation.
- [backend/db/](../backend/db/)
  - `store.go` owns the PostgreSQL pool reference and setup. `users.go`, `events.go`, `tasks.go`, `tags.go`, `sharing.go`, and `timeline.go` group queries by feature. `models.go` and `errors.go` define store types and sentinel errors.
- [backend/db/migrations](../backend/db/migrations)
  - Ordered Goose migrations. Never edit an applied migration; add a new one.
- `handler_test.go` files
  - Testcontainers-backed PostgreSQL integration tests plus handler validation tests.

### Frontend

- [frontend/src/main.tsx](../frontend/src/main.tsx)
  - Creates the React root and the TanStack Query client provider.
- [frontend/src/App.tsx](../frontend/src/App.tsx)
  - Application shell and coordination for workspace, calendar, and drawers. Authentication query and actions live in `frontend/src/features/auth/useAuth.ts`. Event/task mutations live in `frontend/src/features/timeline/useTimelineMutations.ts`. Tag mutations live in `frontend/src/features/tags/useTagMutations.ts`. Calendar state and derived calculations live under `frontend/src/features/calendar/`. Presentational components live under `frontend/src/components/`; shared state helpers live under `frontend/src/hooks/`.
- [frontend/src/App.css](../frontend/src/App.css)
  - Imports feature styles from `frontend/src/styles/`, including tokens, base rules, auth, calendar, sidebar, and drawers.
- [frontend/src/index.css](../frontend/src/index.css)
  - Global reset, fonts, and base element styles.
- [frontend/vite.config.ts](../frontend/vite.config.ts)
  - React plugin and `/api` development proxy.

## Data Model

### Owned timeline items

`timeline_items` is the common owned-item row:

```text
users 1 --- n timeline_items
              |
              +--- events
              +--- tasks
```

`timeline_items.user_id` is the single source of ownership. Event and task subtype rows reference the timeline item by primary key.

Events store exact `starts_at` and `ends_at`, event kind, location, and commute durations. Tasks store only `deadline_at` and `is_completed` in the demo.

### Personal tags

```text
tags 1 --- n timeline_item_tags n --- 1 timeline_items
```

A tag belongs to a user. The database and store queries require both the item and tag to belong to the authenticated user before attaching or removing a tag. Shared-tag adoption is future functionality.

Each user has a `general` tag and a `default_tag_id`. Event creation inserts
the default tag in the same transaction as the event. Every event has a main
tag, and that tag must also be attached to the event. Event updates validate
both tag ownership and attachment. Event tag removal locks the item and refuses
to remove its final tag. When a tag is deleted, events that use it as their main
tag receive a replacement main tag, and events that would otherwise have no
tags receive the replacement tag. Deleting the default tag selects the same
replacement as the new default.

### Shared events

```text
events
  |
  +--- event_shares --- recipient_user_id
             |
             +--- event_share_overrides
```

The event remains owned by the original user's `timeline_items` row. `event_shares` stores the recipient relationship. `event_share_overrides` stores only recipient-local title, description, URL, location, and private custom info.

The owner is derived through `event_shares.event_id -> events.timeline_item_id -> timeline_items.user_id`; it is deliberately not duplicated in `event_shares`.

A shared event is a projection, not a copy. The source fields are read from the owner's current event, and local override fields are applied only for the recipient. This is why the API can show both the source title and the recipient's local title.

## SQL Code Generation

`sqlc` was evaluated for the current PostgreSQL store and is not adopted yet. It supports PostgreSQL, pgx/v5, generated typed query methods, transaction-bound query sets, and type overrides for `google/uuid`. See the [sqlc configuration guide](https://docs.sqlc.dev/en/stable/reference/config.html), [transaction guide](https://docs.sqlc.dev/en/stable/howto/transactions.html), and [type override guide](https://docs.sqlc.dev/en/stable/howto/overrides.html).

The store currently has 26 methods and about 44 pgx query or exec calls. Its files already group SQL by feature. The main benefit from `sqlc` would be generated parameter and result types, with less manual row scanning. Adoption would also require named query files, code generation configuration, generated Go files, and a UUID override to keep `google/uuid`. Several operations still need handwritten transaction and validation workflows around queries. Keep the current pgx methods and review `sqlc` again if the query count or repeated scan code grows. The CLI would be a development tool, not a runtime dependency.

## Request Flow

For an authenticated timeline request:

1. The browser sends the HTTP-only `calendar_session` cookie.
2. The auth handler hashes the raw token and looks up a non-expired session.
3. The timeline handler receives the authenticated user ID through `CurrentUser`.
4. The handler validates request data and calls the store.
5. Store SQL enforces ownership again in the mutation or query.
6. PostgreSQL constraints enforce relationships and basic invariants.
7. The handler returns JSON or an HTTP error.

Authorization is intentionally enforced at the database query boundary as well as in the handler. A guessed UUID must not be enough to edit, complete, tag, or delete another user's data.

## HTTP API

All API routes use JSON request and response bodies unless the response is `204 No Content`. Error responses use plain text. Authenticated routes require the HTTP-only `calendar_session` cookie.

| Method and path | Request JSON | Successful response |
| --- | --- | --- |
| `POST /api/auth/register` | `email`, `password` | `201` with `{ "id": "<uuid>" }` |
| `POST /api/auth/login` | `email`, `password` | `200` with `{ "id": "<uuid>" }` and a session cookie |
| `POST /api/auth/logout` | None | `204`; clears the session cookie |
| `GET /api/auth/me` | None | `{ "id": "<uuid>", "email": "..." }` |
| `GET /api/timeline` | None | Array of timeline entries |
| `POST /api/events` | Event fields below | `201` with `{ "id": "<uuid>" }` |
| `PUT /api/events/{eventID}` | Event fields below | `204` |
| `POST /api/tasks` | Task fields below | `201` with `{ "id": "<uuid>" }` |
| `PUT /api/tasks/{taskID}` | Task fields below | `204` |
| `PATCH /api/tasks/{taskID}` | `is_completed` | `204` |
| `DELETE /api/timeline/{itemID}` | None | `204` |
| `GET /api/tags` | None | Array of tags |
| `POST /api/tags` | `name` | `201` with a tag |
| `PATCH /api/tags/{tagID}` | `name`, `color` | `204` |
| `DELETE /api/tags/{tagID}` | None | `204` |
| `PUT /api/tags/default` | `tag_id` | `204` |
| `POST /api/timeline/{itemID}/tags` | `tag_id` | `204` |
| `DELETE /api/timeline/{itemID}/tags/{tagID}` | None | `204` |
| `POST /api/events/{eventID}/shares` | `recipient_email` | `201` with `{ "id": "<share-uuid>" }` |
| `GET /api/shared-events/{shareID}` | None | Shared event response below |
| `PATCH /api/shared-events/{shareID}` | Any supplied override fields below | `204` |

Event request fields are `title`, `description`, `url`, `starts_at`, `ends_at`, `kind`, `location`, `commute_before_seconds`, `commute_after_seconds`, and `main_tag_id`. `title`, `starts_at`, `ends_at`, and `kind` must be valid. `starts_at` and `ends_at` are timestamps, and the end must be later than the start. The `kind` value is `attendable` or `informational`. Commute durations cannot be negative. If `main_tag_id` is omitted when creating an event, the user's default tag is used. If it is omitted when updating an event, the existing main tag is kept.

Task request fields are `title`, `description`, `url`, and `deadline_at`. The title and deadline are required. `deadline_at` is a timestamp.

Tag colors use six hexadecimal digits with a leading `#`, such as `#547968`. The digits can use uppercase or lowercase letters. The frontend chooses black or white tag text by comparing each color's WCAG contrast ratio against both. Tag responses contain `id`, `name`, `color`, and `is_default`.

Timeline entries contain `id`, `type`, `title`, and `is_completed`. Optional fields are `description`, `url`, `starts_at`, `ends_at`, `deadline_at`, `kind`, `location`, `commute_before_seconds`, `commute_after_seconds`, `is_shared`, `source_title`, `custom_info`, `tags`, and `main_tag_id`. Optional response fields are omitted when empty. Timestamps use JSON time format.

Shared event responses contain `share_id`, `event_id`, `original_title`, `starts_at`, `ends_at`, and `kind`. Optional fields are `original_description`, `original_url`, `original_location`, `local_title`, `local_description`, `local_url`, `local_location`, and `private_custom_info`. The patch accepts any subset of the local fields: `title`, `description`, `url`, `location`, and `custom_info`.

## Frontend Data Flow

The frontend uses:

- `date-fns` for week arithmetic, local formatting, and hour extraction;
- TanStack Query for current-user and workspace server state;
- a small `api<T>()` wrapper for JSON requests and cookie credentials;
- a custom CSS grid for the seven-day week view.

The week grid is not a pre-built calendar component. It derives each item's vertical position from minutes after local midnight and renders static blocks. Dragging is intentionally absent. An item's position changes only after saving an edited event through the API.

The workspace query returns both owned and shared timeline entries. Shared entries already contain effective local presentation values and source metadata, so the UI does not need to reconstruct sharing rules.

## Where to Change Things

| Need | Primary location | Related location |
| --- | --- | --- |
| Add or change an API route | `backend/cmd/server.go` | handler file for the domain |
| Change auth/session behavior | `backend/internal/auth/handler.go` | `backend/db/users.go`, migration 2 |
| Change event/task validation | `backend/internal/timeline/validation.go` | PostgreSQL constraints and feature handler/store files |
| Change event/task persistence | `backend/db/events.go` or `backend/db/tasks.go` | migrations 1 and 3 |
| Change sharing ownership or overrides | `backend/db/sharing.go` and `backend/db/timeline.go` | migrations 4/5, timeline handlers |
| Change tag ownership, defaults, or attachment | `backend/db/tags.go` | migrations 6-8, timeline handlers |
| Change timeline JSON shape | `db.TimelineEntry` in `backend/db/models.go` | frontend `TimelineEntry` type and grid rendering |
| Change week navigation/date math | `frontend/src/App.tsx` date helpers and week state | `date-fns` imports |
| Change event block placement | `frontend/src/App.tsx` `renderGridEntry` | `.grid-entry` in `frontend/src/App.css` |
| Change hour grid dimensions | `frontend/src/App.css` `.week-grid`, `.hour-labels`, `.day-column` | `renderGridEntry` positioning |
| Change create-item fields | composer in `frontend/src/App.tsx` | backend request structs and validation |
| Change client caching/loading | TanStack Query setup in `frontend/src/main.tsx` and workspace query in `App.tsx` | API wrapper |
| Change colors, spacing, or layout | `frontend/src/styles/` | `frontend/src/index.css` for global rules |

## Maintainability Rules

- Keep domain invariants in the backend. The frontend is not an authorization boundary.
- Keep source event data separate from recipient-local shared-event presentation.
- Use transactions when one logical operation updates both `timeline_items` and a subtype row.
- Add a migration for schema changes; do not rewrite applied migrations.
- Prefer typed request/response structs over unstructured maps.
- Keep the custom week-grid calculations in small date/layout helpers. Do not replace them with a full calendar widget unless the product scope changes.
- Use TanStack Query for server state and invalidation rather than adding more manually synchronized arrays.
- Use `date-fns` for date arithmetic rather than mutating `Date` objects by hand.
- Add comments for ownership boundaries, derived projections, and SQL that would be difficult for a beginner to understand. Avoid comments that restate obvious code.

## Recommended Refactor Sequence

The current demo is functional, but `App.tsx` and `store.go` are becoming broad. Refactor in this order when the next feature requires it:

1. Move remaining auth, timeline, tag, and calendar mutations out of `App.tsx` only when this reduces state coordination.
2. Consider a backend service layer only when workflows span multiple stores or need reusable business rules. Do not add one solely for indirection.
3. Add UI and API support for edit/share/local overrides before implementing future scheduling or recurrence features.

## Future Architecture Pressure

The final product will eventually need explicit models for:

- uncertain/date-only/recurring time;
- reminders and notification delivery;
- conflict detection;
- task scheduling sessions and available-time calculation;
- richer per-recipient sharing policies;
- tag adoption and automation triggers.

Those features should extend the current source/projection distinction rather than mutate it. Scheduling should remain derived from events and task sessions, and recipient presentation should remain separate from owner data.

## Verification

From the repository root:

```sh
go test ./... -count=1
go vet ./...
cd frontend && mise exec -- npm run build && mise exec -- npm run lint
```

The PostgreSQL-backed tests use Testcontainers, so Docker must be running.
