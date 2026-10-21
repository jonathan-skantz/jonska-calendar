# HTTP API

- All routes are under `/api`. Bodies are JSON, except error responses, which are plain text.
- Authenticated routes need the `calendar_session` cookie. Login sets it.
- The routes are registered in `backend/cmd/server.go`. The code is the final word.

## Status codes

| Code  | Meaning                                                                                    |
| ----- | ------------------------------------------------------------------------------------------ |
| `400` | Invalid input. The body says why.                                                          |
| `401` | No valid session.                                                                          |
| `404` | Not found **or** not yours. The server does not say which, on purpose.                     |
| `409` | Conflict: duplicate name, already shared, already a member, last or main tag removal.      |
| `429` | Quick add rate limit reached (30 per user per hour).                                       |
| `502` | Quick add: Anthropic failed.                                                               |
| `503` | Not configured or unavailable: quick add without a key, weather with no cache or upstream. |

## Routes

| Method and path                                                         | Request JSON                                          | Success response                                                                     |
| ----------------------------------------------------------------------- | ----------------------------------------------------- | ------------------------------------------------------------------------------------ |
| `POST /api/auth/register`                                               | `email`, `password`                                   | `201` with `{ "id": "<uuid>" }`                                                      |
| `POST /api/auth/login`                                                  | `email`, `password`                                   | `200` with `{ "id": "<uuid>" }` and a session cookie                                 |
| `POST /api/auth/logout`                                                 | None                                                  | `204`; clears the cookie                                                             |
| `GET /api/auth/me`                                                      | None                                                  | `{ "id", "email" }`, plus `display_name` when set                                    |
| `PATCH /api/auth/me`                                                    | `display_name` (empty clears it)                      | `204`                                                                                |
| `GET /api/timeline`                                                     | None                                                  | Array of timeline entries                                                            |
| `POST /api/events`                                                      | Event fields                                          | `201` with `{ "id": "<uuid>" }`                                                      |
| `PUT /api/events/{eventID}`                                             | Event fields                                          | `204`                                                                                |
| `POST /api/tasks`                                                       | Task fields                                           | `201` with `{ "id": "<uuid>" }`                                                      |
| `PUT /api/tasks/{taskID}`                                               | Task fields                                           | `204`                                                                                |
| `PATCH /api/tasks/{taskID}`                                             | `is_completed`                                        | `204`                                                                                |
| `DELETE /api/timeline/{itemID}`                                         | None                                                  | `204`                                                                                |
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
| `GET /api/reminders`                                                    | None                                                  | Your reminders for events that have not ended                                        |
| `POST /api/events/{eventID}/reminders`                                  | `offset_seconds`                                      | `201`; `404` event not visible; `409` offset already set                             |
| `DELETE /api/reminders/{reminderID}`                                    | None                                                  | `204`; `404` not yours                                                               |
| `POST /api/reminders/{reminderID}/notified`                             | None                                                  | `204`; keeps the first delivery time                                                 |
| `GET /api/transit-card/days?from=&to=`                                  | None                                                  | Array of reported transit-card days                                                  |
| `PUT /api/transit-card/days`                                            | `from`, `to`, `needed_dates`, `maybe_dates`           | `204`                                                                                |
| `POST /api/events/parse`                                                | `text` (1 to 500 characters)                          | `{draft, warnings, unresolved}`; `429`, `502`, `503`                                 |
| `POST /api/assistant/messages`                                          | `text`, `time_zone`, `week_start`, `conversation_id?` | `{conversation_id, reply, actions, steps, ended}`; `404`, `409`, `429`, `502`, `503` |
| `POST /api/assistant/conversations/{conversationID}/actions/{actionID}` | `decision` (`apply`, `decline`, `undo`)               | The action with its new `status`; `400`, `404`, `409`                                |
| `GET /api/weather?lat=&lon=`                                            | None                                                  | `{date, conditions, sunrise?, sunset?}[]`; `503`                                     |
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

- The database repeats the commute, gathering, all-day, and skip rules as constraints (migrations 14, 18, 19, 20).
- Durations are seconds in the API. The UI works in minutes and rounds to whole minutes before sending.

## Task fields

- Required: `title`, `deadline_at`. Optional: `description`, `url`.
- The app does not offer tags on tasks. The tag endpoints accept any owned item, so the API would accept tags on a task.

## Share fields

- `title`, `description`, `url`, `location`. Each is trimmed. Empty or missing means "same as the event".
- `PUT` replaces all four.
- Responses contain `id`, `recipient_user_id`, `recipient_email`, `recipient_display_name`, and any share fields set.

## Timeline entries

`GET /api/timeline` returns your owned items and items shared with you.

- Every entry has `id`, `type` (`event` or `task`), `title`, and `is_completed`.
- Other fields appear only when they have a value. They include `kind`, `starts_at`, `ends_at`, `tags`, `main_tag_id`, and the commute and gathering fields.
- `is_shared` and `shared_by` (the owner's display name) appear on items shared with you. Those never include tags, `main_tag_id`, commute, or `is_skipped`.
- `share_count` appears on your own events that you have shared.
- There is no pagination or date filter. The whole timeline is returned each time. See [gotchas](gotchas.md#the-whole-timeline-loads-every-time).

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
