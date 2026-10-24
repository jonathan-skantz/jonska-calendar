# Gotchas

Each entry is a fact about the code, with a pointer to where it lives.

## Authorization and data

### Ownership is checked twice

The handler checks the user and input. The SQL checks `user_id = $caller` on every write. Removing either check is a security bug.

### 404 means "not yours" or "does not exist"

This is deliberate. Someone else's event and a made-up ID look the same. In tests, expect `404`, not `403`.

### Recipients are read-only

A recipient's `PUT`, `DELETE`, tag, and share calls return `404`. Reminders are the exception: recipients set their own. The check is "owner or has a share row" (`canSeeEventSQL` in `db/reminders.go`).

### Shares are not copies

Edits to an event reach recipients unless the owner set a share field. The recipient's view is rebuilt on every load. Do not write projected values into the recipient's data.

### The owner is not stored on `event_shares`

It comes from `events → timeline_items.user_id`. A migration removed the stored copy (migration 5).

### Removing the main tag

- `Store.RemoveTagFromItem` (`backend/db/tags.go`) returns `409` for the event's main tag: "choose another main tag before removing this one".
- To move the main tag, save the event with the new main tag first, then remove the old tag. The composer does this in that order.
- See [data model: tags](data-model.md#tags).

### Skipped events and recipients

- Skipping is private. The owner's skip flag is set only in the owner's view.
- Recipients see `is_skipped` as false, so a recipient's reminder for the event still fires.

## Saving

### A failed save can leave half an event

Creating an event is several requests: the event, each tag, each reminder. There is no rollback. If a later request fails:

- the event already exists, and the form shows an error;
- the workspace may not refresh, because `onSuccess` does not run;
- saving again creates a second event.

Editing is safer: tags are attached first, then the event is saved, then old tags are removed. If the save fails, nothing is lost, and retrying is safe.

Fixing this takes one request or one transaction for the whole save. That is a product decision as well as a code change, so ask the maintainer first.

### Units differ between layers

- The UI works in minutes. The API takes seconds.
- The backend accepts only whole minutes. Sending seconds that are not multiples of 60 returns `400`.

### UI names are not stored names

The form says "Attend" and "Heads-up". The database stores `attendable` and `informational`. Check `kind` when debugging.

### Hidden fields

- Fields inside a hidden section are not in `FormData`.
- The payload builder recomputes dependent flags from `kind` and `is_all_day`, so a hidden commute value does not survive a switch to all-day. Check the builder, not just the form.
- Booleans travel as the strings `"true"` and `"false"`. `buildEventPayload` compares against `"true"`.

## Time

### All-day end is exclusive

- An all-day event on 12 October ends at **00:00 on 13 October**.
- The API only checks that the end is after the start. It does not check for midnight, so a client can store an all-day event that ends at 23:59. Do not rely on the API for this convention.
- The frontend does the conversion (`localDayStart`, `timelineRequests.ts`).

### Times are local for display, UTC on the wire

- You see the browser's time zone. The API uses UTC.
- Sunrise and sunset are `HH:MM` strings with no zone, on purpose.

### Daylight saving shifts the grid

On a clock-change day, items after the change can be off by up to an hour in real elapsed time. Nothing handles this, and no test covers it.

### The transit card rule is UTC-based

The server rejects a `from` earlier than yesterday in UTC, because it does not know your zone. Near midnight, that can reject a day that is "today" for you. The frontend checks the local day first.

### Imports read times in the browser's zone

- Floating times (no `Z`, no `TZID`) and all-day dates in an `.ics` file are read in the `time_zone` the browser sends. The same file imported from another zone lands at other times.
- A `TZID` that is neither an IANA nor a Windows zone name is also read in that zone, with a warning.
- The library would read floating times in the server's zone, so `backend/internal/ical/parse.go` reads the time values itself.

### Imported reminders that already passed count as delivered

`Store.ImportEvents` records a delivery for imported reminders whose time has passed. Otherwise they would all show as "Missed reminder" on the next open.

### Repeating events repeat in their own zone

- A series stores the time zone it was saved in (`recurrence_time_zone`, the browser's) and repeats its wall-clock times there. Viewed from another zone, its occurrences keep the saved zone's local times.
- `events.starts_at` is a series' **first** occurrence. Code that reads events directly must expand series through `timeline.ExpandTimeline` (or `recurrence.Series`); filtering on `starts_at` alone misses later occurrences.
- An occurrence is identified by its event's `id` and its `occurrence_start`, which stays the same when it is moved. The frontend keys entries with `entryKey` (`lib/timeline.ts`).

### Subscriptions cannot reach private addresses

- The subscription client refuses loopback, private, link-local, carrier-grade NAT, and other non-public addresses. It checks the address it dials, so a hostname or a redirect cannot lead it into the server's network.
- A calendar server on your own network (or another copy of this app on `localhost`) is refused. Set `SUBSCRIPTIONS_ALLOW_PRIVATE=true` to allow it, only where every user is trusted.

### Subscriptions refresh in the background

- `go Refresher.Run(ctx)` in `backend/cmd/server.go` runs beside the hourly cleanup job (`backend/internal/cleanup`). Each API process runs one, so two processes refresh the same subscriptions twice; the row lock keeps the results consistent.
- A subscription is refreshed when its last fetch is 6 hours old, checked every 15 minutes. A failed fetch counts as a fetch, so a broken feed is retried after 6 hours, not every 15 minutes.
- Subscribed events are replaced on each refresh and get new IDs. Anything keyed by their ID, such as an open drawer, goes stale after a refresh.

### Feed addresses are secrets

A feed's token is its only credential. Anyone with the address sees the events. Resetting the address (`POST /api/feeds/{id}/token`) is the only way to revoke it.

### Exported series have no VTIMEZONE

`.ics` exports write a series' times with a `TZID` that is an IANA name, without a `VTIMEZONE` component. Google, Apple, and this app read that; a strict RFC 5545 reader may not.

## Setup and tooling

### Rate limits trust `X-Forwarded-For`

Login and registration are limited per client IP, taken from the last `X-Forwarded-For` entry (`backend/internal/middleware/clientip.go`). Caddy overwrites that header, so this is safe only while the API is reachable through Caddy alone. Never publish port 8080 in production. Limits live in memory and reset when the API restarts.

Docker keeps the real client address for IPv4. If the domain gets an IPv6 (`AAAA`) record, enable IPv6 in Docker too; otherwise Docker's proxy hides IPv6 clients behind one address, and they share one limit.

### `goose` does not read `DATABASE_URL`

Goose reads only `GOOSE_*` keys from `.env`. Export `backend/.env` first ([getting started](getting-started.md#environment-files)).

### Run the API from `backend/`

The API loads `.env` and `.env.local` from its working directory. From anywhere else, `DATABASE_URL` is empty.

### Migrations sort by number, not by name

Files are `1_` to `23_`, not zero-padded. Sorting by name puts `10_` before `2_`. Tools and tests use the number. The next file is `24_`.

### Tests and dev use different PostgreSQL versions

- `go test` uses `postgres:15.3-alpine`. Dev uses `postgres:18`.
- A query that depends on behavior that changed between those versions can pass in one and fail in the other.

### There is no CI

- The only automatic check is the pre-commit hook, which runs only after `git config core.hooksPath .githooks`.
- `git commit --no-verify` skips it. Run the checks yourself before you push.

## Not built yet

These are not bugs. [PRODUCT.md](../PRODUCT.md) lists the full set of **Future** items.

- **No resizing by drag.** Dragging moves an item; change its length in the form.
- **No repeating tasks.** Events repeat; tasks do not.
- **No month or agenda view.** The grid shows 7 days, or 3 on phones.
- **Reminders need an open tab.** No push, no background worker. A reminder whose time passed while no tab was open shows as an in-page toast on the next load, not a system notification.
- **No task reminders.** Event reminders exist.
- **No task scheduling.** Tasks get no time slots.
- **Import and export cover events only.** Tasks and shares are not exported, and repeat rules outside the supported ones are skipped. Subscribed events cannot have reminders.
- **No password reset, email verification, password change, or account deletion.**
- **No login throttling, CSRF token, or `Secure` cookie flag.** The cookie is `HttpOnly` and `SameSite=Lax`.
- **No shared tags or tag adoption.**
- **No pagination of tasks.** `GET /api/timeline` returns every task; events are loaded by range.
- **No graceful shutdown.** The server ignores signals.
- **No rate limit on login or register.** Only quick add is limited: 30 per user per hour, in memory, reset on restart.
- **Expired sessions are never deleted.** They are rejected on lookup, but the rows remain.
- **Tag filters are not saved.** The sidebar's tag filter resets on reload.

### Tasks load every time

`GET /api/timeline` returns the events in the range it is asked for, but every task with them. The frontend asks for the visible days and a week on each side. That suits a family calendar; many tasks would need a range too.
