# Technical Architecture

This document describes the current demo architecture and the intended path toward the product in `PRODUCT.md`.

## Scope

The demo is a small family calendar. It currently supports:

- email/password registration, login, logout, and cookie sessions;
- owned events with exact start/end times;
- deadline-only tasks and completion state;
- a custom seven-day, 24-hour React week grid;
- personal tags;
- direct event sharing between registered users, with per-recipient fields set by the owner;
- per-day shared transit-card declarations, households with conflict detection, and user display names.

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
                       (title, description, url, location: owner-set, nullable)
```

The event remains owned by the original user's `timeline_items` row. `event_shares` stores the recipient relationship and the owner's per-recipient values. A NULL field means the recipient sees the event's own value, so later edits to the event still reach them; times, all-day, and kind always come from the event. Recipients cannot override anything. Only the owner manages shares, and every share query checks ownership through `timeline_items.user_id`.

The owner is derived through `event_shares.event_id -> events.timeline_item_id -> timeline_items.user_id`; it is deliberately not duplicated in `event_shares`.

A shared event is a projection, not a copy. Each field is the share's value when the owner set one, and otherwise the owner's current event value. The recipient only ever receives these effective values.

### Transit card declarations

```text
users 1 --- n transit_card_days (user_id, day DATE, need)
users n --- 1 households (via household_members, user_id UNIQUE)
```

Declarations are not timeline items. A row is one user's answer for one local
calendar date: `need` is `yes`, `maybe`, or `no`. A missing row means "not
reported" and `no` is an explicit "not needed". Saving writes a row for every day in the submitted
range in one statement, so an empty selection is still a report. `day` is a
`DATE`, so time zones never shift it.

A household has a name and a `transit_card_count`. Because a user belongs to
at most one household, declarations stay keyed by user and the household is
derived through `household_members`; leaving a household keeps the user's
answers. Household queries never take a household ID from the client, only the
caller's own membership. Listing days returns the caller's rows plus those of
every other member of their household. `status` is computed per request in
`transitCardStatuses` (`backend/internal/timeline/transit_card.go`): a `yes`
row is `conflict` when more members said `yes` that day than the household has
cards. `maybe` never conflicts, and a user without a household never sees a
conflict.

`users.display_name` is optional. Queries read names through the SQL function `user_display_name(display_name, email)` (migration 17), which falls back to the email local-part.

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

| Method and path                                 | Request JSON                                | Successful response                                  |
| ----------------------------------------------- | ------------------------------------------- | ---------------------------------------------------- |
| `POST /api/auth/register`                       | `email`, `password`                         | `201` with `{ "id": "<uuid>" }`                      |
| `POST /api/auth/login`                          | `email`, `password`                         | `200` with `{ "id": "<uuid>" }` and a session cookie |
| `POST /api/auth/logout`                         | None                                        | `204`; clears the session cookie                     |
| `GET /api/auth/me`                              | None                                        | `{ "id", "email" }`, plus `display_name` when set    |
| `PATCH /api/auth/me`                            | `display_name` (empty clears it)            | `204`                                                |
| `GET /api/timeline`                             | None                                        | Array of timeline entries                            |
| `POST /api/events`                              | Event fields below                          | `201` with `{ "id": "<uuid>" }`                      |
| `PUT /api/events/{eventID}`                     | Event fields below                          | `204`                                                |
| `POST /api/tasks`                               | Task fields below                           | `201` with `{ "id": "<uuid>" }`                      |
| `PUT /api/tasks/{taskID}`                       | Task fields below                           | `204`                                                |
| `PATCH /api/tasks/{taskID}`                     | `is_completed`                              | `204`                                                |
| `DELETE /api/timeline/{itemID}`                 | None                                        | `204`                                                |
| `GET /api/tags`                                 | None                                        | Array of tags                                        |
| `POST /api/tags`                                | `name`                                      | `201` with a tag                                     |
| `PATCH /api/tags/{tagID}`                       | `name`, `color`                             | `204`                                                |
| `DELETE /api/tags/{tagID}`                      | None                                        | `204`                                                |
| `PUT /api/tags/default`                         | `tag_id`                                    | `204`                                                |
| `POST /api/timeline/{itemID}/tags`              | `tag_id`                                    | `204`                                                |
| `DELETE /api/timeline/{itemID}/tags/{tagID}`    | None                                        | `204`                                                |
| `GET /api/events/{eventID}/shares`              | None                                        | Array of event shares (owner only)                   |
| `POST /api/events/{eventID}/shares`             | `recipient_email`, optional share fields    | `201` with `{ "id": "<share-uuid>" }`                |
| `PUT /api/events/{eventID}/shares/{shareID}`    | Share fields                                | `204`                                                |
| `DELETE /api/events/{eventID}/shares/{shareID}` | None                                        | `204`                                                |
| `GET /api/transit-card/days?from=&to=`          | None                                        | Array of reported transit-card days                  |
| `PUT /api/transit-card/days`                    | `from`, `to`, `needed_dates`, `maybe_dates` | `204`                                                |
| `GET /api/household`                            | None                                        | Household or `null`                                  |
| `POST /api/household`                           | `name`, `transit_card_count`                | `201`; `409` if already in a household               |
| `PATCH /api/household`                          | `name`, `transit_card_count`                | `204`; `404` without a household                     |
| `POST /api/household/members`                   | `email`                                     | `204`; `404` unknown email; `409` already a member   |
| `DELETE /api/household/members/me`              | None                                        | `204`; deletes the household when it becomes empty   |

Event request fields are `title`, `description`, `url`, `starts_at`, `ends_at`, `is_all_day`, `kind`, `location`, `commute_before_seconds`, `commute_after_seconds`, and `main_tag_id`. `title`, `starts_at`, `ends_at`, and `kind` must be valid. `starts_at` and `ends_at` are timestamps, and the end must be later than the start. For all-day events, the end timestamp is exclusive and starts at midnight after the last displayed date. The `kind` value is `attendable` or `informational`. Commute durations are seconds in whole minutes, from 0 up to 24 hours, and must be 0 unless the event is a timed (not all-day) `attendable` event. Migration 14 enforces the same rules in the database. The week grid draws commute as hatched bands next to the event block. If `main_tag_id` is omitted when creating an event, the user's default tag is used. If it is omitted when updating an event, the existing main tag is kept.

Task request fields are `title`, `description`, `url`, and `deadline_at`. The title and deadline are required. `deadline_at` is a timestamp.

Tag colors use six hexadecimal digits with a leading `#`, such as `#547968`. The digits can use uppercase or lowercase letters. The frontend chooses black or white tag text by comparing each color's WCAG contrast ratio against both. Tag responses contain `id`, `name`, `color`, and `is_default`.

Timeline entries contain `id`, `type`, `title`, and `is_completed`. Optional fields are `description`, `url`, `starts_at`, `ends_at`, `is_all_day`, `deadline_at`, `kind`, `location`, `commute_before_seconds`, `commute_after_seconds`, `is_shared`, `shared_by`, `share_count`, `tags`, and `main_tag_id`. `shared_by` (the owner's display name) appears only on events shared with the caller, which never include tags, `main_tag_id`, or commute. `share_count` appears only on the caller's own shared events. Optional response fields are omitted when empty. Timestamps use JSON time format.

Share fields are `title`, `description`, `url`, and `location`. Each is trimmed, and an empty or missing value means "same as the event"; `PUT` replaces all four. Event share responses contain `id`, `recipient_user_id`, `recipient_email`, and `recipient_display_name`, plus the share fields that are set. Sharing an event the caller does not own, or listing, updating, or removing its shares, returns `404`. Sharing with yourself returns `400`, and sharing twice with the same user returns `409`.

Transit-card dates use `YYYY-MM-DD`. Ranges are inclusive and cover at most 62 days; `needed_dates` and `maybe_dates` must be inside the range, and no date may appear twice across them; every other day in the range is saved as `no`. The server does not know the user's time zone, so it rejects a `from` earlier than yesterday in UTC, and the frontend enforces the exact local day. Day responses contain `date`, `user_id`, `display_name`, `need`, and `status` (`resolved` or `conflict`). Only reported days are returned, for the caller and the other members of their household.

Household responses contain `id`, `name`, `transit_card_count` (1 to 20), and `members`, each with `user_id` and `display_name`.

## Frontend Data Flow

The frontend uses:

- `date-fns` for week arithmetic, local formatting, and hour extraction;
- TanStack Query for current-user and workspace server state;
- a small `api<T>()` wrapper for JSON requests and cookie credentials;
- a custom CSS grid for the seven-day week view.

The week grid is not a pre-built calendar component. It derives each item's vertical position from minutes after local midnight and renders static blocks. Dragging is intentionally absent. An item's position changes only after saving an edited event through the API.

The workspace query returns both owned and shared timeline entries. Shared entries already contain the effective per-recipient values, so the UI does not need to reconstruct sharing rules, and the owner's original values for customized fields never reach the recipient. Shared entries never include the owner's commute durations, because commute is the owner's personal travel.

## Where to Change Things

| Need                                          | Primary location                                                                 | Related location                                       |
| --------------------------------------------- | -------------------------------------------------------------------------------- | ------------------------------------------------------ |
| Add or change an API route                    | `backend/cmd/server.go`                                                          | handler file for the domain                            |
| Change auth/session behavior                  | `backend/internal/auth/handler.go`                                               | `backend/db/users.go`, migration 2                     |
| Change event/task validation                  | `backend/internal/timeline/validation.go`                                        | PostgreSQL constraints and feature handler/store files |
| Change event/task persistence                 | `backend/db/events.go` or `backend/db/tasks.go`                                  | migrations 1 and 3                                     |
| Change sharing or per-recipient fields        | `backend/db/sharing.go` and `backend/db/timeline.go`                             | migrations 4/5/16, `frontend/src/features/sharing/`    |
| Change tag ownership, defaults, or attachment | `backend/db/tags.go`                                                             | migrations 6-8, timeline handlers                      |
| Change transit-card declarations              | `backend/db/transit_card.go`, `backend/internal/timeline/transit_card.go`        | migration 12, `frontend/src/features/transitCard/`     |
| Change households or card conflicts           | `backend/db/households.go`, `backend/internal/timeline/household.go`             | migration 15, `frontend/src/features/household/`       |
| Change timeline JSON shape                    | `db.TimelineEntry` in `backend/db/models.go`                                     | frontend `TimelineEntry` type and grid rendering       |
| Change week navigation/date math              | `frontend/src/App.tsx` date helpers and week state                               | `date-fns` imports                                     |
| Change event block placement                  | `frontend/src/App.tsx` `renderGridEntry`                                         | `.grid-entry` in `frontend/src/App.css`                |
| Change hour grid dimensions                   | `frontend/src/App.css` `.week-grid`, `.hour-labels`, `.day-column`               | `renderGridEntry` positioning                          |
| Change create-item fields                     | composer in `frontend/src/App.tsx`                                               | backend request structs and validation                 |
| Change client caching/loading                 | TanStack Query setup in `frontend/src/main.tsx` and workspace query in `App.tsx` | API wrapper                                            |
| Change colors, spacing, or layout             | `frontend/src/styles/`                                                           | `frontend/src/index.css` for global rules              |

## Maintainability Rules

- Keep domain invariants in the backend. The frontend is not an authorization boundary.
- Keep source event data separate from the owner's per-recipient share fields.
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
3. Keep sharing changes on the owner-managed share model before implementing future scheduling or recurrence features.

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
