import { describe, expect, it } from "vitest";
// Shared with the assistant's busy rules in Go, so the two cannot drift.
import occupiedSpanCases from "./testdata/occupied-spans.json?raw";
import {
  getDefaultEventTagSelection,
  getCommuteSeconds,
  getGathering,
  getMainTag,
  getOccupiedRange,
  isBusy,
  isEventVisible,
  getEntrySource,
} from "./timeline";
import type { TimelineEntry } from "../types";

const tag = (id: string, name = id) => ({
  id,
  name,
  color: "#336699",
  is_default: false,
});

const event = (overrides: Partial<TimelineEntry> = {}): TimelineEntry => ({
  id: "event-1",
  type: "event",
  title: "Event",
  starts_at: "2026-10-15T10:00:00.000Z",
  ends_at: "2026-10-15T11:00:00.000Z",
  is_completed: false,
  ...overrides,
});

describe("timeline behavior", () => {
  it("selects the default event tag", () => {
    const tags = [tag("first"), { ...tag("default"), is_default: true }];
    expect(getDefaultEventTagSelection(tags)).toEqual({
      tagIds: ["default"],
      mainTagId: "default",
    });
    expect(getDefaultEventTagSelection([tag("first")])).toEqual({
      tagIds: [],
      mainTagId: "first",
    });
    expect(getDefaultEventTagSelection([])).toEqual({
      tagIds: [],
      mainTagId: null,
    });
  });

  it("treats only attended events and tasks as busy", () => {
    expect(isBusy(event({ kind: "attendable" }))).toBe(true);
    expect(isBusy(event({ kind: "informational" }))).toBe(false);
    expect(isBusy(event({ kind: "attendable", is_skipped: true }))).toBe(false);
    expect(isBusy({ ...event(), type: "task" })).toBe(true);
  });

  it("hides commute and gathering while an event is skipped", () => {
    const skipped = event({
      kind: "attendable",
      is_skipped: true,
      commute_before_seconds: 1200,
      gathering_seconds: 1800,
    });
    expect(getCommuteSeconds(skipped)).toEqual({ before: 0, after: 0 });
    expect(getGathering(skipped).seconds).toBe(0);
    expect(getCommuteSeconds({ ...skipped, is_skipped: false }).before).toBe(
      1200,
    );
  });

  it("shows tasks regardless of hidden event tags and hides events with a hidden tag", () => {
    expect(isEventVisible(event({ tags: [tag("work")] }), ["work"])).toBe(
      false,
    );
    expect(isEventVisible(event({ tags: [tag("work")] }), ["home"])).toBe(true);
    expect(isEventVisible({ ...event(), type: "task" }, ["work"])).toBe(true);
  });

  it("hides the events of a hidden subscription", () => {
    const school = { id: "school", name: "School", color: "#123456" };
    expect(isEventVisible(event({ subscription: school }), ["school"])).toBe(
      false,
    );
    expect(isEventVisible(event({ subscription: school }), ["work"])).toBe(
      true,
    );
  });

  it("colors a subscribed event by its calendar and names its source", () => {
    const school = { id: "school", name: "School", color: "#123456" };
    expect(getMainTag(event({ subscription: school }))).toEqual({
      ...school,
      is_default: false,
    });
    expect(getEntrySource(event({ subscription: school }))).toBe("School");
    expect(getEntrySource(event({ shared_by: "Ana" }))).toBe("Ana");
    expect(getEntrySource(event())).toBeUndefined();
  });

  it("uses the selected main tag and falls back to the first tag", () => {
    const tags = [tag("first"), tag("main")];
    expect(getMainTag(event({ tags, main_tag_id: "main" }))?.id).toBe("main");
    expect(getMainTag(event({ tags }))?.id).toBe("first");
  });
});

type OccupiedSpanCase = {
  name: string;
  entry: Omit<TimelineEntry, "id" | "title" | "is_completed">;
  busy: boolean;
  start?: string;
  end?: string;
};

describe("occupied time", () => {
  const cases = JSON.parse(occupiedSpanCases) as OccupiedSpanCase[];
  it.each(cases.map((c) => [c.name, c] as const))("%s", (_, c) => {
    const entry: TimelineEntry = {
      id: "event-1",
      title: "Event",
      is_completed: false,
      ...c.entry,
    };
    expect(isBusy(entry)).toBe(c.busy);
    if (!c.busy) return;
    const range = getOccupiedRange(entry);
    expect(range.start.getTime()).toBe(new Date(c.start!).getTime());
    expect(range.end.getTime()).toBe(new Date(c.end!).getTime());
  });
});
