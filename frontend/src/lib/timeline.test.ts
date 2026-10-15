import { describe, expect, it } from "vitest";
import {
  getBackToBackTrim,
  getElapsedOverlayState,
  getEventTagChanges,
  getDefaultEventTagSelection,
  getMainTag,
  isEventVisible,
  buildTimelineItemRequest,
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
  it("selects the default event tag and computes membership changes", () => {
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
    expect(getEventTagChanges(tags, ["default", "new"])).toEqual({
      add: ["new"],
      remove: [{ ...tag("first"), is_default: false }],
    });
  });

  it("builds event and task create and update requests", () => {
    const eventForm = new FormData();
    eventForm.set("title", "Planning");
    eventForm.set("kind", "attendable");
    eventForm.set("starts_at", "2026-10-15T10:00");
    eventForm.set("ends_at", "2026-10-15T11:00");
    eventForm.set("location", "Office");
    const eventRequest = buildTimelineItemRequest(
      "event",
      null,
      eventForm,
      "work",
    );
    expect(eventRequest.path).toBe("/api/events");
    expect(eventRequest.method).toBe("POST");
    expect(eventRequest.body).toMatchObject({
      title: "Planning",
      kind: "attendable",
      location: "Office",
      main_tag_id: "work",
    });
    expect(
      buildTimelineItemRequest("event", "event-1", eventForm, "work"),
    ).toMatchObject({ path: "/api/events/event-1", method: "PUT" });

    const taskForm = new FormData();
    taskForm.set("title", "Follow up");
    taskForm.set("deadline_at", "2026-10-16T17:00");
    expect(
      buildTimelineItemRequest("task", null, taskForm, null),
    ).toMatchObject({
      path: "/api/tasks",
      method: "POST",
      body: { title: "Follow up", deadline_at: expect.any(String) },
    });
    expect(
      buildTimelineItemRequest("task", "task-1", taskForm, null),
    ).toMatchObject({ path: "/api/tasks/task-1", method: "PUT" });
  });

  it("shows tasks regardless of hidden event tags and hides events with a hidden tag", () => {
    expect(isEventVisible(event({ tags: [tag("work")] }), ["work"])).toBe(false);
    expect(isEventVisible(event({ tags: [tag("work")] }), ["home"])).toBe(true);
    expect(isEventVisible({ ...event(), type: "task" }, ["work"])).toBe(true);
  });

  it("uses the selected main tag and falls back to the first tag", () => {
    const tags = [tag("first"), tag("main")];
    expect(getMainTag(event({ tags, main_tag_id: "main" }))?.id).toBe("main");
    expect(getMainTag(event({ tags }))?.id).toBe("first");
  });

  it("trims only exactly adjacent event edges", () => {
    const previous = event({ ends_at: "2026-10-15T10:00:00Z" });
    const current = event({ starts_at: "2026-10-15T10:00:00.000Z" });
    const next = event({ starts_at: "2026-10-15T11:00:00Z" });
    expect(getBackToBackTrim(current, previous, next)).toEqual({
      trimTop: true,
      trimBottom: true,
    });
    expect(
      getBackToBackTrim(
        current,
        event({ ends_at: "2026-10-15T09:59:00Z" }),
        event({ starts_at: "2026-10-15T11:01:00Z" }),
      ),
    ).toEqual({ trimTop: false, trimBottom: false });
    expect(
      getBackToBackTrim({ ...current, type: "task" }, previous, next).trimTop,
    ).toBe(false);
  });

  it("returns past, current, and future elapsed overlay states", () => {
    const today = new Date(2026, 9, 15, 9, 30);
    const label = (value: Date) => `${value.getHours()}:${value.getMinutes()}`;
    expect(getElapsedOverlayState(new Date(2026, 9, 14), today, label)).toEqual({
      kind: "past",
    });
    expect(getElapsedOverlayState(new Date(2026, 9, 15), today, label)).toEqual({
      kind: "today",
      height: "19rem",
      label: "9:30",
    });
    expect(
      getElapsedOverlayState(new Date(2026, 9, 16), today, label),
    ).toEqual({ kind: "none" });
    expect(
      getElapsedOverlayState(
        new Date(2026, 9, 15),
        new Date(2026, 9, 15),
        label,
      ),
    ).toEqual({ kind: "none" });
  });
});
