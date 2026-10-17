A custom calendar with features that fit my workflow.

## Scope

The first demo is for the user and family. It should prioritize a working,
testable product over polish.

- [x] **Demo:** included in the first working version.
- [ ] **Future:** part of the product direction, but not required for the first demo.

## Design

**Common timeline item fields (Demo):**

- [x] title
- [x] description (optional)
- [x] URL (optional)
- [x] events have an exact start date/time and end date/time
- [x] events can be marked all-day and appear above the hourly calendar grid
- [x] tasks have a deadline date/time and no scheduled start/end time
- [ ] reminder behavior is **Future**

**Tags (Demo):**

- [x] Tags are a user's personal way of organizing timeline items.
- [ ] Sharing an item may offer its tags to the recipient, but recipient tag
      adoption is **Future**.
- [x] Every user starts with a `general` tag. Users can rename or remove it when
      another tag exists.
- [x] Each user chooses a default tag. New events receive that tag automatically
      and can receive additional tags.
- [x] Every event has one main tag selected from its tags. New events use the
      default tag as the initial main tag. The main tag controls the event color.
- [x] Event edges show the main tag color and additional tag colors in order.
- [x] Clicking tags in the calendar filters visible events. Multiple selected tags
      show events matching any selected tag.

**Triggers (Future):**

- [ ] Users can specify custom regexes for:
  - [ ] renaming events that were shared with me
  - [ ] setting the background color of timeline items

**Templates (Future):**

- [ ] Users can specify default titles to be applied when setting the first tag.

---

### Timeline Item Type 1: Events

Events have a start date/time and end date/time.

- [x] A timed event's start and/or end can be marked as approximate (e.g. "about
      10:00 to about 18:00"). The event still occupies its stated times, but each
      approximate edge fades out in the week view and its time is shown as `~10:00`.
      All-day events cannot be approximate. Recipients of a shared event see the
      same approximate edges.

**Sharing events (Demo):**

- [x] Sharing is direct user-to-user. Both users must be registered.
- [x] User A can share an event with users B and C.
- [x] The owner can set the title, description/details, URL, and location
      separately for each recipient (e.g. B sees "Title for B", C sees
      "Title for C"). An empty field shows the event's own value, so later
      edits to the event still reach that recipient. Times, all-day, and kind
      always come from the event.
- [x] Tags are never shared. Recipients see shared events without the owner's
      tags or the owner's original values for fields customized for them.
- [x] Recipients see shared events read-only, marked with the owner's name,
      and cannot edit them.
- [ ] Recipient-local overrides of shared fields are **Future**.
- [ ] Busy-only sharing and independently hiding or transforming fields are
      **Future** unless explicitly added to the demo scope.

**Misc features:**

- [ ] The user can add custom info in a separate field for events that are shared with them, e.g.:
  - `CUSTOM INFO: This lecture will be given on zoom: zoom.com/my-link/...`
  - `CUSTOM INFO: The company's URL: the-company.com/...`
- [ ] Recipient-private custom info is **Future**.
- [ ] Making an event appear in a TODO-list without being completeable is
      **Future**.

#### Type 1.a: Events to be attended (Demo)

- [x] Can have either physical or online location.

- [x] An attendable event can include commute time before and after the event. The
      commute occupies calendar space, but must be visually distinct from the event
      itself so it is obvious that the event has not started during the commute.

- [ ] Gathering time is **Future**. When supported, it must be distinct from the
      actual event start time. Examples include:
  - football game: gather 30 mins before event start
  - career fair: mingle 30 mins before
  - a-kvart

#### Type 1.b: Events not to be attended (informational events, reminders, heads-ups) (Demo)

- [ ] Some events may be purely informational. They should be visually distinct from
      attendable events. Examples include:
  - "new phone released",
  - "deadline for X announced",
  - "I will be at the office on Tuesday"
  - birthdays,
  - name days,
  - holidays,

### Timeline Item Type 2: Tasks

**UI fields (Demo):**

- [x] title
- [x] description (optional)
- [x] URL (optional)
- [x] deadline date/time
- [x] is completed

- [ ] Tasks with estimated duration, scheduled work sessions, priorities, richer
      statuses, event attachments, or automatic placement into free time are
      **Future**. Conflict detection and automatic task scheduling are also
      **Future**.

- [ ] The ability to prevent task completion until a starting date/time is **Future**.

### Shared transit card (Demo)

A household shares one or more physical transit cards. Declarations are
separate from events and tasks.

- [x] Each user answers yes, maybe, or no for each day. Selections do not carry
      over; a day without an answer is "not reported".
- [x] Saving the declaration for the visible week records every remaining day:
      days without a yes or maybe are explicitly "no", so an empty selection is
      still a report.
- [x] Days that have ended are read-only.
- [x] Every week shows a "Card" row, separate from all-day events. Each day
      shows a pill with a bus icon and the user's display name: dashed outline =
      not reported yet, struck-through bus = not needed, green = needed, orange
      = maybe. Days that ended without an answer have no pill. Clicking anywhere in the row opens the declaration editor.
- [x] Users can set a display name in Settings; the email local-part is the
      fallback.
- [x] Users form a household (a family or group). A user belongs to at most
      one household. Any member can add registered users by email, set how
      many transit cards the household shares, or leave.
- [x] The "Card" row still shows only the user's own pill. It turns red when
      the user said yes and more members said yes that day than the household
      has cards. "Maybe" never causes a conflict.
- [x] The declaration editor shows the other members' answers for each day,
      and warns before saving when the user's choice would need more cards
      than the household has.
- [ ] Removing other members, invitations that must be accepted, and several
      households per user are **Future**.

## Future Features

The following product ideas are intentionally deferred from the first demo:

- [ ] conflict detection and conflict resolution
- [ ] reminders and notification delivery
- [ ] gathering time as a separate schedule concept
- [ ] recurring events, including birthdays, name days, and holidays
- [ ] external calendar integration or import
- [ ] automatic task scheduling and scheduled task sessions
- [ ] task priorities, richer statuses, estimated duration, and event attachments
- [ ] busy-only sharing and advanced per-field sharing policies
- [ ] recipient tag adoption workflows
- [ ] regex triggers and title/color automation
- [ ] title templates
- [ ] backup, import/export, and audit/history features
- [ ] connect commute time to SL API or other transit data to show expected arrival times and delays

---
