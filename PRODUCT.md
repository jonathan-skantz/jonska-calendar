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
- [x] events can have any number of reminders, each a time before the event
      starts (e.g. 15 min, 1 day; for all-day events, e.g. the day before at
      09:00). Reminders move with the event.
  - Reminders are private: owners and recipients each set their own on any
    event they can see, and they are never shared. Removing a share removes the
    recipient's reminders.
  - They are not drawn in the week view; events with reminders show a small 🔔
    before the title. They are edited in the event's details (or the
    read-only view of a shared event), and all of them are listed in a
    reminders drawer from the 🔔 button in the top bar for quick removal.
  - While the app is open, a reminder shows as a browser notification (or an
    in-app message without permission). Reminders that came due while the app
    was closed show once as "Missed reminder" on the next open. Each reminder
    is delivered once across tabs and devices; moving the event re-arms it.
  - Skipped events don't remind.
- [ ] Task reminders are **Future**.

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

- [x] An attendable event can include gathering time before the event starts,
      with an optional short label (e.g. "Warm-up", "Mingle"). Gathering is
      distinct from the actual event start and from the hatched commute:
  - Mandatory gathering (e.g. warm up with the team) is part of the event block,
    with a line marking the real start. It counts as busy time.
  - Optional gathering (e.g. mingle from 17:00) is a dotted band above the event.
    It only counts as busy time when the event has a commute before, because
    then the user plans to be there.
  - Commute before comes before gathering. Gathering describes the event, so
    recipients of a shared event see it too. Examples include:
  - football game: gather 30 mins before event start
  - career fair: mingle 30 mins before
  - a-kvart

- [x] An attendable event can be marked as skipped (e.g. one lecture the user won't
      go to) while staying in the calendar. It keeps its kind, commute, and
      gathering, but no longer takes up time: it is drawn like a heads-up with a
      ⊘ marker and a dashed outline, and its commute and gathering are hidden.
      Skipping is the owner's own decision and is not shared.
- [ ] Recipients marking a shared event as skipped for themselves is **Future**.

#### Type 1.b: Events not to be attended (informational events, reminders, heads-ups) (Demo)

- [x] Some events may be purely informational. The UI calls the two kinds
      "Attend" (the user takes part, in person, online, or at home) and
      "Heads-up" (something to know about); new events default to Attend.
      Heads-ups are drawn as outlined ghost blocks with an ⓘ marker, while
      attended events are solid. Heads-ups don't take up time: attended events
      are laid out as if they weren't there and draw on top of them. Examples
      include:
  - "new phone released",
  - "deadline for X announced",
  - "I will be at the office on Tuesday"
  - birthdays,
  - name days,
  - holidays,

#### Quick add (Demo)

- [x] A field next to "New" takes one sentence in Swedish or English, e.g.
      "middag hos mormor på lördag ca 18–21 #familj, påminn en timme innan".
      Pressing Enter opens the new-event form with the stated fields filled in:
      title, date and time, all-day, approximate edges, location, attend or
      heads-up, existing tags, commute, gathering, and reminders.
- [x] Nothing is saved until the user reviews the form and presses Save.
- [x] Fields the sentence doesn't mention keep the form's defaults. Parts that
      can't be pinned down (e.g. "sometime next week"), tags that don't exist,
      a missing date or time, or a past date are listed above the form instead
      of being guessed. Tags are never created from quick add.
- [x] "Friday" means the next Friday (today included); "next Friday" means
      Friday in the following week, by the user's week start.
- [x] The sentence is sent to Anthropic's API to be read. Each user can do 30
      quick adds per hour.

#### Assistant (Demo)

A conversation for requests that take more than one step, e.g. "I might eat
lunch with my friend on Friday. If I'm busy, I'll do it next week on Monday
instead".

- [x] An "Assistant" button in the top bar opens a drawer with a conversation.
      The assistant decides for itself what to look up: it lists events and
      finds free time in the user's own calendar, including events shared with
      them, and keeps going until it can answer or suggest something.
- [x] Free time follows the week view: heads-ups and skipped events are free;
      commute, mandatory gathering, and attended all-day events are busy;
      optional gathering is busy only after a commute. This is the only part
      of conflict detection in the demo.
- [x] It creates new events right away. Each one shows as a card with Open
      (edit it in the event form) and Undo (delete it).
- [x] It can move, skip, unskip, or delete the user's own events, but only
      asks: each change shows as a card with Apply and Decline, and nothing
      changes until the user applies it. A change is applied to the event as
      it is then and checked like any edit. Events shared with the user are
      never changed.
- [x] The assistant hears what the user applied, declined, or undid with
      their next message, along with what still waits for them.
- [x] Each answer lists what the assistant checked. It may ask a follow-up
      question; the conversation keeps its context for 30 minutes or 10
      messages, and its cards work for those 30 minutes even after it can
      take no more messages. Each user can send 20 messages per hour.
- [ ] Sharing from the assistant (always confirmed), weather-aware
      suggestions, undoing applied changes, and fitting tasks into free time
      are **Future**.

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

### Weather

- [x] Each day heading in the week view shows one icon per weather condition
      seen in daylight that day (sunny, partly cloudy, cloudy, fog, rain, snow,
      thunder). A sunny day with showers shows both a sun and rain.
- [x] A condition must last at least two daylight hours to be shown (thunder:
      one hour). Cloudy is left out when it rains or snows. At most three icons
      are shown, keeping the most disruptive weather.
- [x] The forecast is for the browser's current location. Without location
      permission, or outside the forecast window (the past 7 days to 16 days
      ahead), no icons are shown.
- [x] Forecasts come from Open-Meteo and are cached on the server per area of
      about 10 km for 3 hours, shared by all users in that area.
- [x] Each day column shows a yellow line with a sun and the local time at
      sunrise and at sunset, and shades the night (midnight to sunrise and
      sunset to midnight) darker. These use the same forecast and window.
- [x] Settings has separate toggles for sunrise lines, sunset lines, and night
      shading; all are on by default and saved per user in the browser.
- [x] Days where the sun does not rise or set (polar day or night) show no
      line for the missing event, and no shading when both are missing.

### Week view

- [x] Zoom fits 24, 16, or 12 hours on screen; the rest scrolls. The choice
      is saved per user in the browser.
- [x] A Today button returns to the range with today, and today's date is
      marked in the day headings.
- [x] Short or narrow blocks show a shorter label; hovering any block shows
      its full title, time, and location.

## Future Features

The following product ideas are intentionally deferred from the first demo:

- [ ] conflict detection and conflict resolution
- [ ] task reminders, and reminder delivery while the app is closed (web push or
      a phone app)
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
