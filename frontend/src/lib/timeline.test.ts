import { describe, expect, it } from "vitest";
import { getEventAdjacency } from "../features/calendar/timelineView";
import { gridOffset } from "./calendar";
import {
  getElapsedOverlayState,
  getEventTagChanges,
  getDefaultEventTagSelection,
  getCommuteSeconds,
  getGathering,
  getMainTag,
  isBusy,
  isEventVisible,
  buildEventPayload,
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

  it("sends commute seconds only for timed attendable events", () => {
    const form = new FormData();
    form.set("title", "Football");
    form.set("kind", "attendable");
    form.set("starts_at", "2026-10-15T18:00");
    form.set("ends_at", "2026-10-15T19:30");
    form.set("commute_before_minutes", "20");
    form.set("commute_after_minutes", "");
    expect(buildEventPayload(form)).toMatchObject({
      commute_before_seconds: 1200,
      commute_after_seconds: 0,
    });

    form.set("commute_before_minutes", "7.6");
    expect(buildEventPayload(form).commute_before_seconds).toBe(480);

    form.set("kind", "informational");
    expect(buildEventPayload(form).commute_before_seconds).toBe(0);

    form.set("kind", "attendable");
    form.set("is_all_day", "true");
    expect(buildEventPayload(form).commute_before_seconds).toBe(0);
  });

  it("sends gathering seconds only for timed attendable events", () => {
    const form = new FormData();
    form.set("title", "Football");
    form.set("kind", "attendable");
    form.set("starts_at", "2026-10-15T18:00");
    form.set("ends_at", "2026-10-15T19:30");
    form.set("gathering_minutes", "30");
    form.set("gathering_is_optional", "true");
    form.set("gathering_label", "  Mingle ");
    expect(buildEventPayload(form)).toMatchObject({
      gathering_seconds: 1800,
      gathering_is_optional: true,
      gathering_label: "Mingle",
    });

    form.set("gathering_minutes", "");
    expect(buildEventPayload(form)).toMatchObject({
      gathering_seconds: 0,
      gathering_is_optional: false,
      gathering_label: null,
    });

    form.set("gathering_minutes", "30");
    form.set("kind", "informational");
    expect(buildEventPayload(form).gathering_seconds).toBe(0);
  });

  it("sends approximate edges only for timed events", () => {
    const form = new FormData();
    form.set("title", "Workshop");
    form.set("starts_at", "2026-10-15T10:00");
    form.set("ends_at", "2026-10-15T18:00");
    form.set("end_is_approximate", "true");
    expect(buildEventPayload(form)).toMatchObject({
      start_is_approximate: false,
      end_is_approximate: true,
    });

    form.set("is_all_day", "true");
    expect(buildEventPayload(form).end_is_approximate).toBe(false);
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

  it("sends skipping only for attendable events, keeping commute", () => {
    const form = new FormData();
    form.set("title", "Lecture");
    form.set("kind", "attendable");
    form.set("starts_at", "2026-10-15T10:00");
    form.set("ends_at", "2026-10-15T12:00");
    form.set("is_skipped", "true");
    form.set("commute_before_minutes", "20");
    expect(buildEventPayload(form)).toMatchObject({
      is_skipped: true,
      commute_before_seconds: 1200,
    });

    form.set("kind", "informational");
    expect(buildEventPayload(form).is_skipped).toBe(false);
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

  it("trims only exactly adjacent event edges", () => {
    const previous = event({ ends_at: "2026-10-15T10:00:00Z" });
    const current = event({ starts_at: "2026-10-15T10:00:00.000Z" });
    const next = event({ starts_at: "2026-10-15T11:00:00Z" });
    expect(getEventAdjacency(current, previous, next)).toEqual({
      trimTop: true,
      trimBottom: true,
    });
    expect(
      getEventAdjacency(
        current,
        event({ ends_at: "2026-10-15T09:59:00Z" }),
        event({ starts_at: "2026-10-15T11:01:00Z" }),
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
