import { describe, expect, it } from "vitest";
import {
  buildEventPayload,
  buildTimelineItemRequest,
  getEventTagChanges,
} from "./timelineRequests";

const tag = (id: string, name = id) => ({
  id,
  name,
  color: "#336699",
  is_default: false,
});

describe("timeline requests", () => {
  it("computes event tag membership changes", () => {
    const tags = [tag("first"), { ...tag("default"), is_default: true }];
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
});
