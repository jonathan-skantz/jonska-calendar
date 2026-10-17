import { describe, expect, it } from "vitest";
import {
  entriesOverlap,
  eventBodyOverlapsDate,
  getCalendarEntryLayout,
  getCommuteSegments,
  getEntriesForDate,
  getEventOverlapDepth,
  getGatheringSegment,
} from "./timelineView";
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
    expect(getEventOverlapDepth(commuting, [earlier])).toBe(1);
    expect(
      getEventOverlapDepth({ ...commuting, commute_before_seconds: 0 }, [
        earlier,
      ]),
    ).toBe(0);
    expect(
      getEventOverlapDepth(commuting, [
        event({ id: "before", ends_at: "2026-10-15T17:40:00" }),
      ]),
    ).toBe(0);
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
    expect(getEventOverlapDepth(ordered[0], [])).toBe(0);
    expect(getEventOverlapDepth(ordered[1], ordered.slice(0, 1))).toBe(1);
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
    expect(getEventOverlapDepth(gathering, [earlier])).toBe(1);
    expect(
      getEventOverlapDepth({ ...gathering, gathering_seconds: 0 }, [earlier]),
    ).toBe(0);
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
      endsAtEventStart: false,
    });
    expect(eventBodyOverlapsDate(early, day)).toBe(true);
    expect(getGatheringSegment(early, new Date(2026, 9, 16))).toEqual({
      minutes: 30,
      isOptional: false,
      label: undefined,
      startMinutes: 0,
      durationMinutes: 15,
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
      expect(getEventOverlapDepth(mingle, [earlier])).toBe(0);
    });

    it("blocks time when there is a commute to it", () => {
      const commuting = { ...mingle, commute_before_seconds: 20 * 60 };
      expect(getCommuteSegments(commuting, day)[0]).toMatchObject({
        position: "before",
        startMinutes: 1000,
        durationMinutes: 20,
      });
      expect(getEventOverlapDepth(commuting, [earlier])).toBe(1);
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
    expect(getEventOverlapDepth(lecture, [])).toBe(0);
  });
});
