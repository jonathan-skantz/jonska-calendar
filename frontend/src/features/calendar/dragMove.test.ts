import { describe, expect, it } from "vitest";
import type { TimelineEntry } from "../../types";
import {
  canDrag,
  getDropStart,
  getGrabOffset,
  moveEntry,
  snapToHalfHour,
  timeFromPoint,
} from "./dragMove";
import type { DragColumn } from "./dragMove";

const event = (overrides: Partial<TimelineEntry> = {}): TimelineEntry => ({
  id: "event-1",
  type: "event",
  title: "Planning",
  starts_at: new Date(2026, 9, 15, 10, 10).toISOString(),
  ends_at: new Date(2026, 9, 15, 11, 10).toISOString(),
  is_completed: false,
  ...overrides,
});

const task: TimelineEntry = {
  id: "task-1",
  type: "task",
  title: "Report",
  deadline_at: new Date(2026, 9, 15, 17, 0).toISOString(),
  is_completed: false,
};

// Two 100px wide columns, 2400px tall: one pixel is 0.6 minutes.
const columns: DragColumn[] = [
  { date: new Date(2026, 9, 15), rect: new DOMRect(0, 0, 100, 2400) },
  { date: new Date(2026, 9, 16), rect: new DOMRect(100, 0, 100, 2400) },
];

describe("canDrag", () => {
  it("allows own timed events and tasks", () => {
    expect(canDrag(event())).toBe(true);
    expect(canDrag(task)).toBe(true);
  });

  it("refuses shared, all-day and untimed items", () => {
    expect(canDrag(event({ shared_by: "Ana" }))).toBe(false);
    expect(canDrag(event({ is_all_day: true }))).toBe(false);
    expect(canDrag(event({ ends_at: undefined }))).toBe(false);
    expect(canDrag({ ...task, deadline_at: undefined })).toBe(false);
  });
});

describe("snapToHalfHour", () => {
  it("rounds to the nearest :00 or :30", () => {
    expect(snapToHalfHour(new Date(2026, 9, 15, 10, 14))).toEqual(
      new Date(2026, 9, 15, 10, 0),
    );
    expect(snapToHalfHour(new Date(2026, 9, 15, 10, 15))).toEqual(
      new Date(2026, 9, 15, 10, 30),
    );
    expect(snapToHalfHour(new Date(2026, 9, 15, 10, 45))).toEqual(
      new Date(2026, 9, 15, 11, 0),
    );
  });
});

describe("timeFromPoint", () => {
  it("maps a point to the time in the column under it", () => {
    expect(timeFromPoint(columns, 150, 1000)).toEqual(
      new Date(2026, 9, 16, 10, 0),
    );
  });

  it("clamps points beside, above or below the grid", () => {
    expect(timeFromPoint(columns, -20, -50)).toEqual(new Date(2026, 9, 15));
    expect(timeFromPoint(columns, 500, 3000)).toEqual(new Date(2026, 9, 17));
    expect(timeFromPoint([], 0, 0)).toBeNull();
  });
});

describe("dropping", () => {
  it("keeps the grab point under the pointer and snaps the start", () => {
    const entry = event();
    // Grabbed 20 minutes into the event.
    const offset = getGrabOffset(entry, new Date(2026, 9, 15, 10, 30));
    expect(getDropStart(new Date(2026, 9, 16, 14, 25), offset)).toEqual(
      new Date(2026, 9, 16, 14, 0),
    );
  });

  it("moves an event and keeps its duration", () => {
    const moved = moveEntry(
      event({ ends_at: new Date(2026, 9, 16, 1, 10).toISOString() }),
      new Date(2026, 9, 17, 22, 0),
    );
    expect(new Date(moved.starts_at!)).toEqual(new Date(2026, 9, 17, 22, 0));
    expect(new Date(moved.ends_at!)).toEqual(new Date(2026, 9, 18, 13, 0));
  });

  it("moves a task's deadline", () => {
    const moved = moveEntry(task, new Date(2026, 9, 16, 9, 30));
    expect(new Date(moved.deadline_at!)).toEqual(new Date(2026, 9, 16, 9, 30));
    expect(moved.title).toBe("Report");
  });
});
