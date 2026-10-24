# HTTP API

- All routes are under `/api`. Bodies are JSON, except error responses, which are plain text.
- Authenticated routes need the `calendar_session` cookie. Login sets it.
- Cross-site requests that change data (anything but `GET`, `HEAD`, `OPTIONS`) get `403`, using Go's `http.CrossOriginProtection` (`backend/internal/middleware/api.go`). Clients without a browser, such as curl, send neither `Sec-Fetch-Site` nor `Origin` and are allowed.
- Request bodies are capped at 64 KB (`backend/internal/middleware`), except the `.ics` import, which allows 6 MB. A body that is too large without a declared length gets `400`.
- The routes are registered in `backend/cmd/server.go`. The code is the final word.

## Status codes

| Code  | Meaning                                                                                                                                                                    |
| ----- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `400` | Invalid input. The body says why.                                                                                                                                          |
| `401` | No valid session.                                                                                                                                                          |
| `403` | Registration is invite-only and the email is not on the list, or a demo account tried to share, use a household, or subscribe.                                             |
| `404` | Not found **or** not yours. The server does not say which, on purpose.                                                                                                     |
| `409` | Conflict: duplicate name, already shared, already a member, last or main tag removal, a subscribed event without a UID.                                                    |
| `413` | The body is over 64 KB, or over 6 MB for an `.ics` import.                                                                                                                 |
| `429` | Rate limit reached: login (20 per client per 15 minutes), registration (5 per client per hour), quick add (30 per user per hour).                                          |
| `502` | Quick add: Anthropic failed. Subscription refresh: the calendar could not be fetched.                                                                                      |
| `503` | Not configured or unavailable: quick add or the assistant without a key or past the daily limit, weather with no cache or upstream, the demo when 200 demo accounts exist. |

## Routes

| Method and path                                                         | Request JSON                                          | Success response                                                                     |
| ----------------------------------------------------------------------- | ----------------------------------------------------- | ------------------------------------------------------------------------------------ |
| `POST /api/auth/register`                                               | `email` (a valid address), `password` (8+ characters) | `201` with `{ "id": "<uuid>" }`                                                      |
| `POST /api/auth/login`                                                  | `email`, `password`                                   | `200` with `{ "id": "<uuid>" }` and a session cookie                                 |
| `POST /api/auth/logout`                                                 | None                                                  | `204`; clears the cookie                                                             |
| `POST /api/auth/demo`                                                   | `time_zone`                                           | `201` with `{ "id" }` and a session cookie for a new demo account (24 hours)         |
| `GET /api/auth/me`                                                      | None                                                  | `{ "id", "email" }`, plus `display_name` when set and `demo_expires_at` for a demo   |
| `PATCH /api/auth/me`                                                    | `display_name` (empty clears it)                      | `204`                                                                                |
| `GET /api/timeline?from=&to=`                                           | None                                                  | Events in the range, each occurrence on its own, and all tasks                       |
| `POST /api/events`                                                      | Event fields                                          | `201` with `{ "id": "<uuid>" }`                                                      |
| `PUT /api/events/{eventID}?scope=&occurrence=`                          | Event fields                                          | `204`; `200` with `{ "id" }` of the new series for `following`                       |
| `POST /api/tasks`                                                       | Task fields                                           | `201` with `{ "id": "<uuid>" }`                                                      |
| `PUT /api/tasks/{taskID}`                                               | Task fields                                           | `204`                                                                                |
| `PATCH /api/tasks/{taskID}`                                             | `is_completed`                                        | `204`                                                                                |
| `DELETE /api/timeline/{itemID}?scope=&occurrence=`                      | None                                                  | `204`                                                                                |
| `GET /api/tags`                                                         | None                                                  | Array of tags (`id`, `name`, `color`, `is_default`)                                  |
| `POST /api/tags`                                                        | `name`                                                | `201` with a tag                                                                     |
| `PATCH /api/tags/{tagID}`                                               | `name`, `color` (both required)                       | `204`                                                                                |
| `DELETE /api/tags/{tagID}`                                              | None                                                  | `204`                                                                                |
| `PUT /api/tags/default`                                                 | `tag_id`                                              | `204`                                                                                |
| `POST /api/timeline/{itemID}/tags`                                      | `tag_id`                                              | `204`                                                                                |
| `DELETE /api/timeline/{itemID}/tags/{tagID}`                            | None                                                  | `204`; `409` for the last tag or the main tag                                        |
| `GET /api/events/{eventID}/shares`                                      | None                                                  | Array of shares (owner only)                                                         |
| `POST /api/events/{eventID}/shares`                                     | `recipient_email`, optional share fields              | `201` with `{ "id": "<share-uuid>" }`                                                |
| `PUT /api/events/{eventID}/shares/{shareID}`                            | Share fields                                          | `204`                                                                                |
| `DELETE /api/events/{eventID}/shares/{shareID}`                         | None                                                  | `204`                                                                                |
| `PUT /api/events/{eventID}/customization`                               | Customization fields                                  | `204`; `404` not shared with you                                                     |
| `DELETE /api/events/{eventID}/customization`                            | None                                                  | `204`; `404` not shared with you                                                     |
| `PUT /api/subscribed-events/{eventID}/customization`                    | Customization fields                                  | `204`; `404` not yours; `409` the feed gives it no UID                               |
| `DELETE /api/subscribed-events/{eventID}/customization`                 | None                                                  | `204`; `404` not yours; `409` the feed gives it no UID                               |
| `GET /api/reminders`                                                    | None                                                  | Your reminders for events that have not ended                                        |
| `POST /api/events/{eventID}/reminders`                                  | `offset_seconds`                                      | `201`; `404` event not visible; `409` offset already set                             |
| `DELETE /api/reminders/{reminderID}`                                    | None                                                  | `204`; `404` not yours                                                               |
| `POST /api/reminders/{reminderID}/notified`                             | `occurrence_start` (optional)                         | `204`; keeps the first delivery time                                                 |
| `GET /api/transit-card/days?from=&to=`                                  | None                                                  | Array of reported transit-card days                                                  |
| `PUT /api/transit-card/days`                                            | `from`, `to`, `needed_dates`, `maybe_dates`           | `204`                                                                                |
| `POST /api/events/parse`                                                | `text` (1 to 500 characters)                          | `{draft, warnings, unresolved}`; `429`, `502`, `503`                                 |
| `POST /api/assistant/messages`                                          | `text`, `time_zone`, `week_start`, `conversation_id?` | `{conversation_id, reply, actions, steps, ended}`; `404`, `409`, `429`, `502`, `503` |
| `POST /api/assistant/conversations/{conversationID}/actions/{actionID}` | `decision` (`apply`, `decline`, `undo`)               | The action with its new `status`; `400`, `404`, `409`                                |
| `GET /api/weather?lat=&lon=`                                            | None                                                  | `{date, conditions, sunrise?, sunset?}[]`; `503`                                     |
| `POST /api/import/ics`                                                  | `ics`, `time_zone`, `tag_id?`, `dry_run`              | `{imported, skipped, warnings}`; `400`, `404` tag not yours, `413`                   |
| `GET /api/export.ics?time_zone=`                                        | None                                                  | Your own events as a `text/calendar` file download                                   |
| `GET /api/subscriptions`                                                | None                                                  | Array of subscriptions                                                               |
| `POST /api/subscriptions`                                               | `url`, `time_zone`, `name?`, `color?`                 | `201` with the subscription; `400` bad or unreachable feed; `409` already subscribed |
| `PATCH /api/subscriptions/{subscriptionID}`                             | `name`, `color` (both required)                       | The subscription                                                                     |
| `POST /api/subscriptions/{subscriptionID}/refresh`                      | None                                                  | The subscription; `502` with why the feed failed                                     |
| `DELETE /api/subscriptions/{subscriptionID}`                            | None                                                  | `204`; its events go with it                                                         |
| `GET /api/feeds`                                                        | None                                                  | Array of feeds                                                                       |
| `POST /api/feeds`                                                       | `name`, `time_zone`, `tag_id?`, `include_alarms`      | `201` with the feed; `404` tag not yours                                             |
| `PATCH /api/feeds/{feedID}`                                             | `name`, `tag_id?`, `include_alarms`                   | The feed; `404` feed or tag not yours                                                |
| `POST /api/feeds/{feedID}/token`                                        | None                                                  | The feed with a new `path`; the old one answers `404`                                |
| `DELETE /api/feeds/{feedID}`                                            | None                                                  | `204`                                                                                |
| `GET /api/feeds/{token}/calendar.ics`                                   | None, **no session**                                  | The feed as `text/calendar`; `404` unknown token                                     |
| `GET /api/household`                                                    | None                                                  | Household, or `null`                                                                 |
| `POST /api/household`                                                   | `name`, `transit_card_count`                          | `201`; `409` if already in a household                                               |
| `PATCH /api/household`                                                  | `name`, `transit_card_count`                          | `204`; `404` without a household                                                     |
| `POST /api/household/members`                                           | `email`                                               | `204`; `404` unknown email; `409` already a member                                   |
| `DELETE /api/household/members/me`                                      | None                                                  | `204`; deletes the household when it becomes empty                                   |

## Event fields

Required: `title`, `starts_at`, `ends_at`, `kind`.

| Field                                             | Rules                                                                                                                 |
| ------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------- |
| `title`                                           | Trimmed. Must not be blank.                                                                                           |
| `starts_at`, `ends_at`                            | Timestamps. `ends_at` must be later than `starts_at`.                                                                 |
| `is_all_day`                                      | If true, `ends_at` is exclusive: midnight after the last day. See [gotchas](gotchas.md#all-day-end-is-exclusive).     |
| `kind`                                            | `attendable` or `informational`.                                                                                      |
| `is_skipped`                                      | Allowed only on `attendable` events.                                                                                  |
| `start_is_approximate`, `end_is_approximate`      | Display only. Must be false on all-day events.                                                                        |
| `commute_before_seconds`, `commute_after_seconds` | Whole minutes, 0 to 24 hours. Must be 0 unless the event is timed and `attendable`.                                   |
| `gathering_seconds`                               | Same rules as commute. Time at the event before `starts_at`.                                                          |
| `gathering_is_optional`                           | Requires `gathering_seconds`.                                                                                         |
| `gathering_label`                                 | Trimmed, at most 40 characters. Blank is stored as NULL. Requires `gathering_seconds`.                                |
| `description`, `url`, `location`                  | Optional. Blank is stored as NULL.                                                                                    |
| `main_tag_id`                                     | On create, defaults to your default tag. On update, omitting it keeps the current main tag. Must be one of your tags. |
| `recurrence`                                      | Optional repeat rule. See [repeating events](#repeating-events). Omitting it makes the event a plain one.             |
| `time_zone`                                       | The browser's IANA time zone. Required with `recurrence`; the times repeat in it.                                     |

- The database repeats the commute, gathering, all-day, and skip rules as constraints (migrations 14, 18, 19, 20).
- Durations are seconds in the API. The UI works in minutes and rounds to whole minutes before sending.

## Task fields

- Required: `title`, `deadline_at`. Optional: `description`, `url`.
- The app does not offer tags on tasks. The tag endpoints accept any owned item, so the API would accept tags on a task.

## Share fields

- `title`, `description`, `url`, `location`. Each is trimmed. Empty or missing means "same as the event".
- `PUT` replaces all four.
- Responses contain `id`, `recipient_user_id`, `recipient_email`, `recipient_display_name`, and any share fields set.

## Customization fields

A recipient's own changes to an event shared with them or from a calendar they subscribe to. Nobody else sees them.

- `title`, `description`, `url`, `location`, and `note` (the recipient's own text) are trimmed. An empty or missing field keeps the received value, so later changes at the source still show.
- `start_shift_seconds` and `end_shift_seconds` move the start and end. They are whole minutes, at most 24 hours either way, and `0` or missing keeps the edge. Shifts are left out on all-day events, and when the shifted event would not end after it starts.
- `commute_before_seconds` and `commute_after_seconds` add the recipient's own commute (whole minutes, up to 24 hours). Owners never share theirs.
- `gathering_seconds` replaces the received gathering with `gathering_is_optional` and `gathering_label` (at most 40 characters); `0` removes it and a missing value keeps it. Commute and gathering only apply to timed attend events.
- `PUT` replaces all of them. It applies to every occurrence of a repeating event.
- A subscribed event is customized by its `id` in the timeline. The change is kept by the feed's UID, so it survives refreshes and covers every event the feed gives that UID.

## Timeline entries

`GET /api/timeline` returns your owned items and items shared with you.

- Every entry has `id`, `type` (`event` or `task`), `title`, and `is_completed`.
- Other fields appear only when they have a value. They include `kind`, `starts_at`, `ends_at`, `tags`, `main_tag_id`, and the commute and gathering fields.
- `is_shared` and `shared_by` (the owner's display name) appear on items shared with you. Those never include tags, `main_tag_id`, commute, or `is_skipped`.
- `share_count` appears on your own events that you have shared.
- `customization` (the [customization fields](#customization-fields)) appears on received events you customized. They show your customized values, and `original` holds `{title, description?, url?, location?, starts_at, ends_at, gathering_seconds?, gathering_is_optional?, gathering_label?}` as you received them.
- `from` and `to` are required RFC 3339 instants, at most 92 days apart. Events that overlap the range are returned; tasks are all returned. See [gotchas](gotchas.md#tasks-load-every-time).
- Each occurrence of a repeating event is its own entry with the event's `id`, its own `starts_at` and `ends_at`, `occurrence_start` (the start its rule gives it, which identifies it even when moved), and `recurrence` (the rule).

## Repeating events

- `recurrence` is `{ frequency, interval, weekdays?, monthly_by?, until?, count? }`:
  - `frequency` is `daily`, `weekly`, `monthly`, or `yearly`, and `interval` is 1 to 99.
  - `weekdays` (`MO` to `SU`) is for weekly rules and must include the start's weekday; it defaults to it.
  - `monthly_by` is `day_of_month` (default), `weekday` (the start's weekday as the 1st to 4th of the month), or `last_weekday`.
  - `until` (`YYYY-MM-DD`, the last date an occurrence may start on) or `count` (1 to 999) ends it. Neither means it never ends.
- `PUT /api/events/{id}` and `DELETE /api/timeline/{id}` take `scope` (`this`, `following`, or `all`) and `occurrence` (an `occurrence_start`) to change some occurrences of a series:
  - `this` moves or skips one occurrence; any other change answers `400`. Deleting `this` cancels the occurrence.
  - `following` ends the series before the occurrence. On `PUT` it starts a new series with the request's fields and answers its id; a `count` the request keeps goes on for the occurrences that are left.
  - `all` changes the whole series. The series moves as the occurrence moved, keeping its local times, and a weekly series moves to the weekdays the occurrence moved to.
  - Without `scope`, `PUT` saves the series as stored: its times are the first occurrence's.
- `400` for an invalid rule or scope, or a scope on an event that does not repeat; `404` for an `occurrence` that the series does not have.

## Transit card

- Dates are `YYYY-MM-DD`. Ranges are inclusive and cover at most 62 days.
- `needed_dates` and `maybe_dates` must fall inside the range, and no date may appear in both.
- Every other day in the range is saved as `no`.
- The server does not know your time zone. It rejects a `from` earlier than yesterday in UTC. The frontend checks the exact local day.
- Responses contain `date`, `user_id`, `display_name`, `need`, and `status` (`resolved` or `conflict`). Only reported days are returned, for you and your household.

## Household

- Responses contain `id`, `name`, `transit_card_count` (1 to 20), and `members` (`user_id`, `display_name`).
- You add a member by email. There is no invitation step.

## Weather

- `GET /api/weather?lat=&lon=` returns one item per day.
- Each item has `date`, `conditions`, and optional `sunrise` and `sunset` (`HH:MM`, local time at that place).
- Results are cached per 0.1-degree cell. See [weather cache](data-model.md#weather-cache).

## Import and export

- The code is in `backend/internal/ical`. Only `VEVENT`s are read and written; tasks and attendees are not.
- Repeating events are read and written with RRULE, EXDATE (cancelled occurrences), and RECURRENCE-ID events (moved, cancelled, or skipped occurrences). A changed occurrence's other changes are dropped with a warning.
- `POST /api/import/ics` takes the file's text in `ics` and the browser's IANA `time_zone`, used for floating times and all-day dates.
  - `tag_id` is the main tag of every imported event. Leave it out for your default tag. `CATEGORIES` that match your tag names, ignoring case, are attached too. Tags are never created.
  - `dry_run: true` answers what an import would do and writes nothing. The real import runs in one transaction.
  - `skipped` lists `{title, reason}` for events not imported: an unsupported repeat rule (RDATE, EXRULE, or a rule outside the ones above), a changed occurrence whose series is not in the file, cancelled, no start time, an unreadable time, failed event validation (the reason is the validation message), or `already imported`.
  - `warnings` lists details that were dropped, such as a reminder after the start or an unknown time zone.
  - A file holds at most 5000 events. A file that is not iCalendar answers `400`.
- `GET /api/export.ics` returns your own events, not events shared with you, with your tags as `CATEGORIES` and your reminders as `VALARM`s. Fields only this app has are written as `X-JONSKA-*` properties, so importing the file again restores them. An event with gathering has `DTSTART` at the gathering and `X-JONSKA-STARTS-AT-GATHERING:TRUE`, so other apps show the whole event; its alarms, `EXDATE`s, and `RECURRENCE-ID`s are as early. A series whose gathering would start the day before keeps its own start, since its rule's weekdays depend on it.
- See [data model: iCalendar UIDs](data-model.md#icalendar-uids) for how repeated imports are detected.

## Subscriptions

- The code is in `backend/internal/subscription`. See [data model: subscriptions](data-model.md#subscriptions).
- `url` is `http`, `https`, or `webcal` (fetched as `https`), at most 2048 characters. The server fetches it before saving, so `400` says why a feed does not work: unreachable, an error status, over 10 MB, not iCalendar, or a private address.
- `time_zone` is the browser's IANA zone. The feed's floating times and all-day dates are read in it, as on import.
- `name` defaults to the calendar's `NAME` or `X-WR-CALNAME`, else the address's host. `color` is `#rrggbb` and defaults to `#547968`.
- Responses contain `id`, `name`, `url`, `color`, `last_fetched_at`, `last_success_at`, and `last_error` when the last refresh failed.
- Subscribed events are in `GET /api/timeline` with `subscription: {id, name, color}` and no tags, commute, `is_skipped`, or `main_tag_id`. Event routes answer `404` for them, as for any event that is not yours.

## Feeds

- A feed publishes your own events, all or those with `tag_id`, like `GET /api/export.ics`. Your commute and skips are left out, and your reminders are `VALARM`s only with `include_alarms`. Events shared with you and subscribed events are never included.
- Responses contain `id`, `name`, `tag_id` when set, `include_alarms`, and `path`, the address relative to the app (`/api/feeds/<token>/calendar.ics`). The browser adds its own origin.
- `time_zone` is read on create only. All-day dates and plain times are written in it, because a feed request has no browser.
- `GET /api/feeds/{token}/calendar.ics` needs no cookie: the token is the credential. It names the calendar (`NAME`, `X-WR-CALNAME`), asks apps to refresh hourly (`REFRESH-INTERVAL`, `X-PUBLISHED-TTL`), and sets `Cache-Control: private, max-age=300`.

## Quick add

- `POST /api/events/parse` returns a draft, not a saved event.
- `warnings` lists anything that could not be kept. `unresolved` lists what could not be worked out.
- Nothing is written. The user saves through `POST /api/events`.

## Assistant

- `POST /api/assistant/messages` sends one message. Leave out `conversation_id` to start a conversation; send the returned one to continue it.
- `text` is 1 to 1000 characters. `time_zone` is the browser's IANA zone and `week_start` is 0 (Sunday) to 6, because only the browser knows them.
- `actions` are what the assistant did or asks to do: `{id, kind, status, event_id, title, starts_at, ends_at, is_all_day, warnings?}`. `kind` is `create`, `move`, `skip`, `unskip` or `delete`; the times are the event's after the action.
  - `create` runs at once with `status: "done"`; the decision `undo` deletes the event.
  - The others have `status: "pending"` and change nothing until the decision `apply`. `decline` drops them.
  - Each action is decided once: `409` after that. A failed `apply` (`404` the event is gone, `400` the change breaks an event rule or the new time has passed) leaves it pending.
- `steps` lists the tools the assistant ran, as `{tool, summary}`.
- `ended: true` means the conversation takes no more messages (step limit or refusal); its actions can still be decided. `404` means it expired or belongs to someone else, and `409` means it is still answering or is full.
