import { describe, expect, it } from "vitest";
import {
  entriesOverlap,
  eventBodyOverlapsDate,
  getCalendarEntryLayout,
  getCommuteSegments,
  getEntriesForDate,
  getElapsedOverlayState,
  getEventAdjacency,
  getHorizontalPlacements,
  getEventPreview,
  getGatheringSegment,
  getScrollToHour,
  layoutDayEntries,
} from "./timelineView";
import { gridOffset } from "../../lib/calendar";
import type { TimelineEntry } from "../../types";

const event = (overrides: Partial<TimelineEntry> = {}): TimelineEntry => ({
  id: "event-1",
  type: "event",
  title: "Event",
  starts_at: "2026-10-15T10:00:00",
  ends_at: "2026-10-15T11:00:00",
  is_completed: false,
  ...overrides,
});

describe("calendar entry layout", () => {
  it("uses the event duration instead of the minimum visual height", () => {
    expect(
      getCalendarEntryLayout(
        event({ ends_at: "2026-10-15T10:15:00" }),
        new Date(2026, 9, 15, 9),
      ),
    ).toEqual({
      startMinutes: 600,
      durationMinutes: 15,
      fullyElapsed: false,
      continuesFromPreviousDay: false,
      continuesToNextDay: false,
    });
  });

  it("uses the configured task display duration and event completion time", () => {
    const task: TimelineEntry = {
      id: "task-1",
      type: "task",
      title: "Task",
      deadline_at: "2026-10-15T17:00:00",
      is_completed: false,
    };
    expect(getCalendarEntryLayout(task, new Date(2026, 9, 15, 18))).toEqual({
      startMinutes: 1020,
      durationMinutes: 90,
      fullyElapsed: false,
      continuesFromPreviousDay: false,
      continuesToNextDay: false,
    });
    expect(
      getCalendarEntryLayout(event(), new Date(2026, 9, 15, 11)).fullyElapsed,
    ).toBe(true);
  });
});

describe("calendar entries by date", () => {
  it("matches event start dates and task deadlines", () => {
    const task: TimelineEntry = {
      id: "task-1",
      type: "task",
      title: "Task",
      deadline_at: "2026-10-15T17:00:00",
      is_completed: false,
    };
    const otherDay = event({
      id: "other-day",
      starts_at: "2026-10-16T10:00:00",
      ends_at: "2026-10-16T11:00:00",
    });
    expect(
      getEntriesForDate([event(), task, otherDay], new Date(2026, 9, 15)),
    ).toEqual([event(), task]);
  });
});

describe("commute segments", () => {
  const commuting = event({
    kind: "attendable",
    starts_at: "2026-10-15T18:00:00",
    ends_at: "2026-10-15T19:30:00",
    commute_before_seconds: 20 * 60,
    commute_after_seconds: 15 * 60,
  });

  it("returns nothing without commute", () => {
    expect(getCommuteSegments(event(), new Date(2026, 9, 15))).toEqual([]);
  });

  it("places commute before and after the event", () => {
    expect(getCommuteSegments(commuting, new Date(2026, 9, 15))).toEqual([
      {
        position: "before",
        minutes: 20,
        startMinutes: 1060,
        durationMinutes: 20,
      },
      {
        position: "after",
        minutes: 15,
        startMinutes: 1170,
        durationMinutes: 15,
      },
    ]);
  });

  it("clips commute that crosses midnight to each day", () => {
    const earlyEvent = event({
      kind: "attendable",
      starts_at: "2026-10-16T00:10:00",
      ends_at: "2026-10-16T01:00:00",
      commute_before_seconds: 30 * 60,
    });
    const previousDay = new Date(2026, 9, 15);

    expect(getCommuteSegments(earlyEvent, previousDay)).toEqual([
      {
        position: "before",
        minutes: 30,
        startMinutes: 1420,
        durationMinutes: 20,
      },
    ]);
    expect(getCommuteSegments(earlyEvent, new Date(2026, 9, 16))).toEqual([
      { position: "before", minutes: 30, startMinutes: 0, durationMinutes: 10 },
    ]);
    expect(getEntriesForDate([earlyEvent], previousDay)).toEqual([earlyEvent]);
    expect(eventBodyOverlapsDate(earlyEvent, previousDay)).toBe(false);
  });

  it("counts commute as occupied time when overlapping", () => {
    const earlier = event({ id: "earlier", ends_at: "2026-10-15T17:50:00" });
    expect(entriesOverlap(earlier, commuting)).toBe(true);
    expect(
      entriesOverlap(earlier, { ...commuting, commute_before_seconds: 0 }),
    ).toBe(false);
    expect(
      entriesOverlap(
        event({ id: "before", ends_at: "2026-10-15T17:40:00" }),
        commuting,
      ),
    ).toBe(false);
  });

  it("ignores commute on informational and all-day events", () => {
    const day = new Date(2026, 9, 15);
    expect(
      getCommuteSegments({ ...commuting, kind: "informational" }, day),
    ).toEqual([]);
    expect(getCommuteSegments({ ...commuting, is_all_day: true }, day)).toEqual(
      [],
    );
  });

  it("is not elapsed until the commute after has ended", () => {
    const date = new Date(2026, 9, 15);
    expect(
      getCalendarEntryLayout(commuting, new Date(2026, 9, 15, 19, 40), date)
        .fullyElapsed,
    ).toBe(false);
    expect(
      getCalendarEntryLayout(commuting, new Date(2026, 9, 15, 19, 45), date)
        .fullyElapsed,
    ).toBe(true);
  });

  it("orders a day by where each entry visually starts", () => {
    const short = event({
      id: "short",
      starts_at: "2026-10-15T10:00:00",
      ends_at: "2026-10-15T10:30:00",
    });
    const longCommute = event({
      id: "long-commute",
      kind: "attendable",
      starts_at: "2026-10-15T11:00:00",
      ends_at: "2026-10-15T12:00:00",
      commute_before_seconds: 90 * 60,
    });
    const ordered = getEntriesForDate(
      [short, longCommute],
      new Date(2026, 9, 15),
    );
    expect(ordered.map((entry) => entry.id)).toEqual(["long-commute", "short"]);
    expect(entriesOverlap(ordered[0], ordered[1])).toBe(true);
  });
});

describe("gathering", () => {
  const gathering = event({
    kind: "attendable",
    starts_at: "2026-10-15T18:00:00",
    ends_at: "2026-10-15T20:00:00",
    gathering_seconds: 30 * 60,
    commute_before_seconds: 20 * 60,
  });
  const day = new Date(2026, 9, 15);

  it("starts the event block at gathering and commute before it", () => {
    expect(getCalendarEntryLayout(gathering, day, day)).toMatchObject({
      startMinutes: 1050,
      durationMinutes: 150,
    });
    expect(getGatheringSegment(gathering, day)).toEqual({
      minutes: 30,
      isOptional: false,
      label: undefined,
      startMinutes: 1050,
      durationMinutes: 30,
      continuesFromPreviousDay: false,
      endsAtEventStart: true,
    });
    expect(getCommuteSegments(gathering, day)[0]).toMatchObject({
      position: "before",
      startMinutes: 1030,
      durationMinutes: 20,
    });
  });

  it("counts gathering as occupied time when overlapping", () => {
    const earlier = event({ id: "earlier", ends_at: "2026-10-15T17:20:00" });
    expect(entriesOverlap(earlier, gathering)).toBe(true);
    expect(
      entriesOverlap(earlier, { ...gathering, gathering_seconds: 0 }),
    ).toBe(false);
  });

  it("clips gathering that crosses midnight to each day", () => {
    const early = event({
      kind: "attendable",
      starts_at: "2026-10-16T00:15:00",
      ends_at: "2026-10-16T01:00:00",
      gathering_seconds: 30 * 60,
    });
    expect(getGatheringSegment(early, day)).toEqual({
      minutes: 30,
      isOptional: false,
      label: undefined,
      startMinutes: 1425,
      durationMinutes: 15,
      continuesFromPreviousDay: false,
      endsAtEventStart: false,
    });
    expect(eventBodyOverlapsDate(early, day)).toBe(true);
    expect(getGatheringSegment(early, new Date(2026, 9, 16))).toEqual({
      minutes: 30,
      isOptional: false,
      label: undefined,
      startMinutes: 0,
      durationMinutes: 15,
      continuesFromPreviousDay: true,
      endsAtEventStart: true,
    });
  });

  describe("optional", () => {
    const mingle = event({
      kind: "attendable",
      starts_at: "2026-10-15T17:30:00",
      ends_at: "2026-10-15T20:00:00",
      gathering_seconds: 30 * 60,
      gathering_is_optional: true,
      gathering_label: "Mingle",
    });
    const earlier = event({ id: "earlier", ends_at: "2026-10-15T17:15:00" });

    it("keeps the event block at the real start and does not block time", () => {
      expect(getCalendarEntryLayout(mingle, day, day)).toMatchObject({
        startMinutes: 1050,
        durationMinutes: 150,
      });
      expect(getGatheringSegment(mingle, day)).toMatchObject({
        isOptional: true,
        label: "Mingle",
        startMinutes: 1020,
        durationMinutes: 30,
      });
      expect(entriesOverlap(earlier, mingle)).toBe(false);
    });

    it("blocks time when there is a commute to it", () => {
      const commuting = { ...mingle, commute_before_seconds: 20 * 60 };
      expect(getCommuteSegments(commuting, day)[0]).toMatchObject({
        position: "before",
        startMinutes: 1000,
        durationMinutes: 20,
      });
      expect(entriesOverlap(earlier, commuting)).toBe(true);
    });

    it("still shows on the previous day when it crosses midnight", () => {
      const early = event({
        ...mingle,
        starts_at: "2026-10-16T00:10:00",
        ends_at: "2026-10-16T01:00:00",
      });
      expect(getEntriesForDate([early], day)).toEqual([early]);
      expect(eventBodyOverlapsDate(early, day)).toBe(false);
      expect(getGatheringSegment(early, day)).toMatchObject({
        startMinutes: 1420,
        durationMinutes: 20,
        endsAtEventStart: false,
      });
    });
  });

  it("ignores gathering on informational events", () => {
    expect(
      getGatheringSegment({ ...gathering, kind: "informational" }, day),
    ).toBeNull();
  });
});

describe("not-busy layer", () => {
  it("detects overlap between a busy event and a heads-up", () => {
    const lecture = event({ id: "lecture", kind: "attendable" });
    const release = event({
      id: "release",
      kind: "informational",
      starts_at: "2026-10-15T10:30:00",
      ends_at: "2026-10-15T11:30:00",
    });
    expect(entriesOverlap(lecture, release)).toBe(true);
    expect(
      entriesOverlap(lecture, {
        ...release,
        starts_at: "2026-10-15T11:00:00",
      }),
    ).toBe(false);
    // Laid out on its own layer, the lecture is not indented by the heads-up.
    expect(getHorizontalPlacements([lecture])).toEqual([
      { left: 0, right: 1, raised: false },
    ]);
  });
});

describe("new event preview", () => {
  it("ends the default duration after the start", () => {
    expect(
      getEventPreview("2026-10-15T09:00:00", new Date(2026, 9, 15), "24-hour"),
    ).toMatchObject({ startsAt: "09:00", endsAt: "11:00" });
  });

  it("only shows on the day the draft starts", () => {
    expect(
      getEventPreview("2026-10-15T09:00:00", new Date(2026, 9, 16)),
    ).toBeNull();
  });
});

// Exact adjacency compares instants, so these fixtures use UTC times.
const utcEvent = (overrides: Partial<TimelineEntry> = {}) =>
  event({
    starts_at: "2026-10-15T10:00:00.000Z",
    ends_at: "2026-10-15T11:00:00.000Z",
    ...overrides,
  });

describe("adjacency and elapsed overlay", () => {
  it("trims only exactly adjacent event edges", () => {
    const previous = utcEvent({ ends_at: "2026-10-15T10:00:00Z" });
    const current = utcEvent({ starts_at: "2026-10-15T10:00:00.000Z" });
    const next = utcEvent({ starts_at: "2026-10-15T11:00:00Z" });
    expect(getEventAdjacency(current, previous, next)).toEqual({
      trimTop: true,
      trimBottom: true,
    });
    expect(
      getEventAdjacency(
        current,
        utcEvent({ ends_at: "2026-10-15T09:59:00Z" }),
        utcEvent({ starts_at: "2026-10-15T11:01:00Z" }),
      ),
    ).toEqual({ trimTop: false, trimBottom: false });
    expect(
      getEventAdjacency({ ...current, type: "task" }, previous, next).trimTop,
    ).toBe(false);
  });

  it("returns past, current, and future elapsed overlay states", () => {
    const today = new Date(2026, 9, 15, 9, 30);
    const label = (value: Date) => `${value.getHours()}:${value.getMinutes()}`;
    expect(getElapsedOverlayState(new Date(2026, 9, 14), today, label)).toEqual(
      {
        kind: "past",
      },
    );
    expect(getElapsedOverlayState(new Date(2026, 9, 15), today, label)).toEqual(
      {
        kind: "today",
        height: gridOffset(9.5 * 60),
        label: "9:30",
      },
    );
    expect(getElapsedOverlayState(new Date(2026, 9, 16), today, label)).toEqual(
      { kind: "none" },
    );
    expect(
      getElapsedOverlayState(
        new Date(2026, 9, 15),
        new Date(2026, 9, 15),
        label,
      ),
    ).toEqual({ kind: "none" });
  });
});

describe("day entry layout", () => {
  const day = new Date(2026, 9, 15);

  it("lays out busy entries and ghosts as separate layers", () => {
    const meeting = event({ id: "meeting", kind: "attendable" });
    const headsUp = event({ id: "heads-up", kind: "informational" });
    const allDay = event({
      id: "all-day",
      is_all_day: true,
      starts_at: "2026-10-15T00:00:00",
      ends_at: "2026-10-16T00:00:00",
    });

    const { ghosts, busy } = layoutDayEntries(
      [meeting, headsUp, allDay],
      day,
      new Date(2026, 9, 15, 8),
    );

    expect(ghosts.map((item) => item.entry.id)).toEqual(["heads-up"]);
    expect(busy.map((item) => item.entry.id)).toEqual(["meeting"]);
    // The busy block is inset so the overlapped ghost's edge stays visible,
    // and ghosts never count toward the busy layer's overlap depth.
    expect(busy[0]).toMatchObject({
      overlapsGhost: true,
      placement: { left: 0, right: 1, raised: false },
    });
    expect(ghosts[0].overlapsGhost).toBe(false);
  });

  it("marks entries that have fully elapsed", () => {
    const morning = event({
      id: "morning",
      kind: "attendable",
      starts_at: "2026-10-15T08:00:00",
      ends_at: "2026-10-15T09:00:00",
    });
    const later = event({ id: "later", kind: "attendable" });

    const { busy } = layoutDayEntries(
      [morning, later],
      day,
      new Date(2026, 9, 15, 9, 30),
    );

    expect(busy.map((item) => [item.entry.id, item.fullyElapsed])).toEqual([
      ["morning", true],
      ["later", false],
    ]);
  });
});

describe("horizontal placement", () => {
  const busy = (id: string, start: string, end: string) =>
    event({
      id,
      kind: "attendable",
      starts_at: `2026-10-15T${start}:00`,
      ends_at: `2026-10-15T${end}:00`,
    });
  const edges = (layer: TimelineEntry[]) =>
    getHorizontalPlacements(layer).map(({ left, right, raised }) => [
      +left.toFixed(3),
      +right.toFixed(3),
      raised,
    ]);

  it("keeps entries that overlap nothing at full width", () => {
    expect(
      edges([busy("a", "09:00", "10:00"), busy("b", "10:00", "11:00")]),
    ).toEqual([
      [0, 1, false],
      [0, 1, false],
    ]);
  });

  it("puts events that start close together side by side", () => {
    expect(
      edges([
        busy("a", "09:00", "11:00"),
        busy("b", "09:00", "10:00"),
        busy("c", "09:59", "10:00"),
      ]),
    ).toEqual([
      [0, 0.333, false],
      [0.333, 0.667, true],
      [0.667, 1, true],
    ]);
  });

  it("cascades an event that starts later over the earlier one", () => {
    expect(
      edges([
        busy("workshop", "09:00", "13:00"),
        busy("call", "10:00", "12:00"),
        busy("quick", "11:00", "11:30"),
      ]),
    ).toEqual([
      [0, 1, false],
      [0.4, 1, true],
      [0.64, 1, true],
    ]);
  });

  it("splits a cascaded group into columns", () => {
    expect(
      edges([
        busy("workshop", "09:00", "13:00"),
        busy("call", "10:00", "11:00"),
        busy("lunch", "10:15", "11:00"),
      ]),
    ).toEqual([
      [0, 1, false],
      [0.4, 0.7, true],
      [0.7, 1, true],
    ]);
  });

  it("keeps the left part of every overlapped column visible", () => {
    expect(
      edges([
        busy("a", "10:00", "12:00"),
        busy("b", "10:00", "12:00"),
        busy("c", "10:00", "12:00"),
        busy("e", "11:00", "13:00"),
        busy("f", "12:30", "14:00"),
      ]),
    ).toEqual([
      [0, 0.333, false],
      [0.333, 0.667, true],
      [0.667, 1, true],
      // Inside C, clear of the left part of A, B and C.
      [0.8, 1, true],
      // Only E is left to keep visible, so F takes the free space beside it.
      [0, 0.8, false],
    ]);
  });

  it("reuses a column that is free by the time an event starts", () => {
    expect(
      edges([
        busy("short", "10:00", "10:30"),
        busy("long", "10:00", "12:00"),
        busy("next", "10:45", "11:15"),
      ]),
    ).toEqual([
      [0, 0.5, false],
      [0.5, 1, true],
      [0, 0.5, false],
    ]);
  });

  it("only cascades over groups the event overlaps", () => {
    expect(
      edges([
        busy("workshop", "09:00", "13:00"),
        busy("call", "10:00", "11:00"),
        busy("later", "12:00", "12:30"),
      ]),
    ).toEqual([
      [0, 1, false],
      [0.4, 1, true],
      [0.4, 1, true],
    ]);
  });

  it("groups by where commute starts", () => {
    const commuting = {
      ...busy("commuting", "10:00", "11:00"),
      commute_before_seconds: 60 * 60,
    };
    expect(edges([commuting, busy("meeting", "09:15", "10:00")])).toEqual([
      [0, 0.5, false],
      [0.5, 1, true],
    ]);
  });

  it("leaves tasks at full width", () => {
    const task: TimelineEntry = {
      id: "task",
      type: "task",
      title: "Task",
      deadline_at: "2026-10-15T09:00:00",
      is_completed: false,
    };
    expect(edges([busy("a", "09:00", "10:00"), task])).toEqual([
      [0, 1, false],
      [0, 1, false],
    ]);
  });
});

describe("scroll position", () => {
  const dates = [new Date(2026, 9, 15), new Date(2026, 9, 16)];

  it("keeps the previous hour visible when today is shown", () => {
    expect(getScrollToHour(dates, new Date(2026, 9, 16, 14, 20))).toBe(13);
    // Early in the day the default hour still applies.
    expect(getScrollToHour(dates, new Date(2026, 9, 16, 3))).toBe(7);
  });

  it("uses the default hour for ranges without today", () => {
    expect(getScrollToHour(dates, new Date(2026, 9, 20, 14))).toBe(7);
  });
});
