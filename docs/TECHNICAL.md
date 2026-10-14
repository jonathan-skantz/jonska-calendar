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
- [backend/internal/timeline/handler.go](../backend/internal/timeline/handler.go)
  - HTTP decoding, validation, ownership-aware calls, and JSON responses for events, tasks, tags, timeline listing, and sharing.
- [backend/db/store.go](../backend/db/store.go)
  - PostgreSQL queries and transaction boundaries.
  - It currently contains user/session, timeline, sharing, tag, and projection queries. It is the first backend file to split when the domain grows further.
- [backend/db/migrations](../backend/db/migrations)
  - Ordered Goose migrations. Never edit an applied migration; add a new one.
- `handler_test.go` files
  - Testcontainers-backed PostgreSQL integration tests plus handler validation tests.

### Frontend

- [frontend/src/main.tsx](../frontend/src/main.tsx)
  - Creates the React root and the TanStack Query client provider.
- [frontend/src/App.tsx](../frontend/src/App.tsx)
  - Current application shell, auth screen, workspace, custom week grid, composer, and item actions.
  - This is intentionally compact for the demo, but it is now the main frontend refactoring target.
- [frontend/src/App.css](../frontend/src/App.css)
  - All current application styles, including the grid geometry.
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
the default tag in the same transaction as the event. Event tag removal locks
the item and refuses to remove its final tag; deleting a tag first reassigns
otherwise-untagged events to another user tag.

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
| Change auth/session behavior | `backend/internal/auth/handler.go` | auth methods in `backend/db/store.go`, migration 2 |
| Change event/task validation | `backend/internal/timeline/handler.go` | PostgreSQL constraints and store method |
| Change event/task persistence | `backend/db/store.go` | migrations 1 and 3 |
| Change sharing ownership or overrides | `backend/db/store.go` | migration 4/5, timeline handler |
| Change tag ownership, defaults, or attachment | `backend/db/store.go` | migrations 6-8, timeline handler |
| Change timeline JSON shape | `db.TimelineEntry` in `backend/db/store.go` | frontend `TimelineEntry` type and grid rendering |
| Change week navigation/date math | `frontend/src/App.tsx` date helpers and week state | `date-fns` imports |
| Change event block placement | `frontend/src/App.tsx` `renderGridEntry` | `.grid-entry` in `frontend/src/App.css` |
| Change hour grid dimensions | `frontend/src/App.css` `.week-grid`, `.hour-labels`, `.day-column` | `renderGridEntry` positioning |
| Change create-item fields | composer in `frontend/src/App.tsx` | backend request structs and validation |
| Change client caching/loading | TanStack Query setup in `frontend/src/main.tsx` and workspace query in `App.tsx` | API wrapper |
| Change colors, spacing, or layout | `frontend/src/App.css` | `frontend/src/index.css` for global rules |

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

1. Split the frontend into `api/`, `features/auth/`, `features/timeline/`, and `components/`.
2. Move the week grid into a `WeekView` component and keep date/layout helpers beside it.
3. Move the composer into an `ItemComposer` component and share typed event/task form models.
4. Split `store.go` into `auth_store.go`, `timeline_store.go`, `sharing_store.go`, and `tag_store.go` while keeping `db.Store` as the shared connection owner.
5. Add a small backend service layer only when workflows span multiple stores or need reusable business rules. Do not add one solely for indirection.
6. Add UI and API support for edit/share/local overrides before implementing future scheduling or recurrence features.

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
