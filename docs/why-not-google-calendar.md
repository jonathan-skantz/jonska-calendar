# Why this app exists

How Jonska Calendar differs from Google Calendar, and why I built it that way. Statements about Google Calendar are as of October 2026.

## The short answer

The app is built around how my family plans a day. It shows the real time a day takes, lets each person see a view written for them, and puts weather, light, and card needs in view while planning.

## Core differences

### A. Shape of an event

| This app                                                                                                                            | Why                             | Google Calendar                                                                                                                                                                                                                                                         |
| ----------------------------------------------------------------------------------------------------------------------------------- | ------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Approximate start and end times (`~10:00`, faded edge).                                                                             | Times are often fuzzy.          | Doesn't support approximate or fuzzy times.                                                                                                                                                                                                                             |
| Commute before and after, as hatched bands that take up time.                                                                       | I want to see when I'm busy.    | Can't show commute or travel time as a band that takes up time.                                                                                                                                                                                                         |
| Gathering before an event. Mandatory gathering is part of the block. Optional gathering is a dotted band, busy only with a commute. | I want to see when I'm busy.    | No equivalent found in Google's help pages.                                                                                                                                                                                                                             |
| A skipped event stays visible but takes no time. Its commute and gathering are hidden. Skipping is private.                         | I want to keep what I skip.     | Declining sends a response email to the organizer and appears in the guest list ([Invite people](https://support.google.com/calendar/answer/37161)). A setting shows events you said "No" to ([Calendar settings](https://support.google.com/calendar/answer/6084644)). |
| "Attend" and "Heads-up". Heads-ups are ghost blocks that take no time.                                                              | Knowing differs from attending. | No equivalent found in Google's help pages. Each event has a Busy or Free setting ([Event visibility](https://support.google.com/calendar/answer/34580)).                                                                                                               |

### B. Sharing

| This app                                                                                                           | Why                                                                                        | Google Calendar                                                                                                                                                                      |
| ------------------------------------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| The owner sets a title, details, URL, and location for each recipient. An empty field shows the event's own value. | I want a different view per person, without copies. Copies add clutter and go out of sync. | Per-event guest permissions: "Modify event", "Invite others", "See guest list" ([Share events](https://support.google.com/calendar/answer/13839531)). No per-guest title or details. |
| Tags, colors, commute, and the skipped state are never shared.                                                     | My organization stays mine.                                                                | Guests' reminders and calendar settings are their own ([Invite people](https://support.google.com/calendar/answer/37161)).                                                           |
| A shared event is a view of the owner's event, not a copy. Edits reach everyone.                                   | Edits should reach everyone without copies.                                                | Guests can edit only when "Modify event" is on ([Share events](https://support.google.com/calendar/answer/13839531)).                                                                |

Recipients see shared events read-only, marked with the owner's name.

### C. Household and transit card

| This app                                                                                                                                            | Why                                                                              | Google Calendar                   |
| --------------------------------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------- | --------------------------------- |
| Each person answers yes, maybe, or no per day. The household sets its card count. A warning appears when more members say yes than there are cards. | My household has several people and fewer cards. A card is not an event or task. | No equivalent in Google Calendar. |

### D. Awareness and speed

| This app                                                                                                  | Why                                      | Google Calendar                                                                                                                                                                                   |
| --------------------------------------------------------------------------------------------------------- | ---------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Weather icons per day, sunrise and sunset lines, and night shading.                                       | I want to plan around weather and light. | Google Calendar doesn't show weather or sunrise and sunset natively. Third-party calendars do, so this isn't unique. This app doesn't need a third-party calendar.                                |
| Quick add: one sentence in Swedish or English opens a prefilled form. Nothing saves until you press Save. | I want to add events in one sentence.    | Quick create takes a title and time ([Create an event](https://support.google.com/calendar/answer/72143)). Google's Assistant can also create events from speech. I haven't confirmed its limits. |

## Parity, not a difference

**Personal reminders.** Google also gives each guest their own reminders: "Your Google Calendar guests receive reminders for your event based on their own settings" ([Invite people](https://support.google.com/calendar/answer/37161)). This app does the same. The extras:

- reminders move with the event;
- a skipped event does not remind its owner, but recipients still get reminders, because skipping is not shared;
- each reminder is delivered once across tabs and devices.

## Other differences

- Recipients cannot edit a shared event.
- Card answers are per day, with no carry-over.
- The card warning appears when more members say yes than there are cards.
- Events and tasks can be dragged to a new time, snapping to the half hour, but not resized.
- A task shows its deadline as a fixed block at that time. Google's Deadline is day-only and appears in the All day section ([Tasks in Calendar](https://support.google.com/calendar/answer/9901136)).
- An event can have several tags, with one main tag that sets its color. Google sets an event's color directly ([Event color](https://support.google.com/calendar/answer/15619910)) and offers labels for tracking time ([Labels](https://support.google.com/calendar/answer/12377581)).

## Features this app does not have yet

These are on the **Future** list in [PRODUCT.md](../PRODUCT.md):

- recurring events;
- importing or connecting external calendars;
- task reminders;
- reminders while the app is closed (web push or a phone app);
- conflict detection between events;
- backup, export, and history.

## Sources

- [Share events from your Google Calendar](https://support.google.com/calendar/answer/13839531)
- [Invite people to your Calendar event](https://support.google.com/calendar/answer/37161)
- [Create an event](https://support.google.com/calendar/answer/72143)
- [Change event and task visibility](https://support.google.com/calendar/answer/34580)
- [Create and manage tasks in Google Calendar](https://support.google.com/calendar/answer/9901136)
- [Change Google Calendar settings](https://support.google.com/calendar/answer/6084644)
- [Change event color set and density](https://support.google.com/calendar/answer/15619910)
- [Labels](https://support.google.com/calendar/answer/12377581)
