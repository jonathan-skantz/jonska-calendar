# First week

A suggested path for your first few days. It is a guide, not a checklist.

## Day 1: run it and use it

1. Follow [getting started](getting-started.md) until you can register, add an event, and see it on the grid.
2. Add a task and tick it off. Watch the grid and the task list update.
3. Register a second account in a private window. Share an event with it, then log in as that account and see the shared copy.

## Day 2: trace one request

- Follow [walkthrough 1](walkthroughs.md#walkthrough-1-create-an-event) in the code, with a breakpoint or `log` line in each function it names.
- Repeat with [walkthrough 2](walkthroughs.md#walkthrough-2-share-an-event).

## Day 3: read the tests

- `backend/internal/timeline/handler_test.go`: a handler tested against a real database.
- `frontend/src/features/timeline/useTimelineMutations.test.tsx`: a hook tested with a mocked API.
- Run the suite and watch it pass. Then break one assertion on purpose and watch it fail.

## Days 4 and 5: a small change

Pick one. Each touches a clear place.

- **Add a test** for a rule that has none. Look at `validation.go` and the frontend `lib/` helpers.
- **Fix a wrong message.** Find an error string that does not match the code, and correct it. Add a test if the string is checked.
- **Check one doc claim.** Pick a statement in these docs, check it against the code, and fix whichever is wrong. Tell the maintainer what you changed.

Ask the maintainer before you change behavior. [PRODUCT.md](../PRODUCT.md) decides what is correct.

**TODO(maintainer):** replace this section with real starter tasks, tickets, or links.

## Glossary

| Term                        | Meaning                                                                                      |
| --------------------------- | -------------------------------------------------------------------------------------------- |
| **Attendable**              | An event you must attend. Counts as busy time unless skipped. The UI says "Attend".          |
| **Informational**           | An event on the calendar that takes no time. The UI says "Heads-up".                         |
| **All-day**                 | An event with no time. Ends at midnight after its last day.                                  |
| **Approximate edge**        | A rough start or end time. Drawn faded and marked `~`. Display only.                         |
| **Commute**                 | Travel before or after an event. Personal, never shared.                                     |
| **Gathering**               | Time at the event before it starts. Shared, because it describes the event. May be optional. |
| **Ghost**                   | A faint entry that is not busy: heads-up, skipped, or a task.                                |
| **Main tag**                | An event's primary tag. Must be one of the event's tags.                                     |
| **Default tag**             | The tag a new event gets when none is chosen.                                                |
| **Owner**                   | The user whose `timeline_items` row holds an event. Found through `user_id`.                 |
| **Recipient**               | A user an event is shared with. Read-only, except for their own reminders.                   |
| **Share**                   | A row that lets a recipient see an event. The recipient's view is built on read.             |
| **Projection**              | A view built from source data at read time, not a stored copy. Shares are projections.       |
| **Reminder**                | A private alert, set as an offset before an event starts. Fires while a tab is open.         |
| **Household**               | A group of users sharing one transit card. A user belongs to at most one.                    |
| **Transit-card day**        | One user's answer for one date: `yes`, `maybe`, or `no`.                                     |
| **Conflict (transit card)** | More household members said `yes` than there are cards that day. Computed per request.       |
| **Quick add**               | Typing one sentence to get a draft event, using the Anthropic API. Saves nothing by itself.  |
| **Goose**                   | The migration tool. Runs the numbered SQL files.                                             |
| **Testcontainers**          | A library that starts a Docker container for tests. Here, PostgreSQL.                        |
| **Future**                  | A feature in `PRODUCT.md` that is not built and is out of scope unless asked for.            |
