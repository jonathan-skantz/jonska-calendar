import { describe, expect, it } from "vitest";
import {
  getDefaultEventTagSelection,
  getCommuteSeconds,
  getGathering,
  getMainTag,
  isBusy,
  isEventVisible,
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

  it("uses the selected main tag and falls back to the first tag", () => {
    const tags = [tag("first"), tag("main")];
    expect(getMainTag(event({ tags, main_tag_id: "main" }))?.id).toBe("main");
    expect(getMainTag(event({ tags }))?.id).toBe("first");
  });
});
