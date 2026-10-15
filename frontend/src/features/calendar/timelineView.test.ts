import { describe, expect, it } from "vitest";
import { getCalendarEntryLayout } from "./timelineView";
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
  it("keeps event duration at the minimum visual height", () => {
    expect(
      getCalendarEntryLayout(
        event({ ends_at: "2026-10-15T10:15:00" }),
        new Date(2026, 9, 15, 9),
      ),
    ).toEqual({ startMinutes: 600, durationMinutes: 90, fullyElapsed: false });
  });

  it("uses the configured task display duration and event completion time", () => {
    const task: TimelineEntry = {
      id: "task-1",
      type: "task",
      title: "Task",
      deadline_at: "2026-10-15T17:00:00",
      is_completed: false,
    };
    expect(
      getCalendarEntryLayout(task, new Date(2026, 9, 15, 18)),
    ).toEqual({ startMinutes: 1020, durationMinutes: 90, fullyElapsed: false });
    expect(
      getCalendarEntryLayout(
        event(),
        new Date(2026, 9, 15, 11),
      ).fullyElapsed,
    ).toBe(true);
  });
});
