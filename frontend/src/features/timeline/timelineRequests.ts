import { SECONDS_PER_MINUTE, localDateTime } from "../../lib/calendar";
import type { Tag } from "../../types";

type EventPayload = {
  title: string;
  description: string | null;
  url: string | null;
  kind: "attendable" | "informational";
  is_skipped: boolean;
  starts_at: string | undefined;
  ends_at: string | undefined;
  is_all_day: boolean;
  start_is_approximate: boolean;
  end_is_approximate: boolean;
  location: FormDataEntryValue | null;
  gathering_seconds: number;
  gathering_is_optional: boolean;
  gathering_label: string | null;
  commute_before_seconds: number;
  commute_after_seconds: number;
};

type TaskPayload = {
  title: string;
  description: string | null;
  url: string | null;
  deadline_at: string | undefined;
};

export const getEventTagChanges = (previous: Tag[], nextTagIds: string[]) => {
  const previousTagIds = new Set(previous.map((tag) => tag.id));
  const nextIDs = new Set(nextTagIds);
  return {
    add: nextTagIds.filter((tagId) => !previousTagIds.has(tagId)),
    remove: previous.filter((tag) => !nextIDs.has(tag.id)),
  };
};

type TimelineItemRequest = {
  path: string;
  method: "POST" | "PUT";
  body:
    | EventPayload
    | TaskPayload
    | (EventPayload & { main_tag_id: string | null });
};

const localDayStart = (value: FormDataEntryValue | null, nextDay = false) => {
  if (typeof value !== "string" || !value) return undefined;
  const date = new Date(`${value.slice(0, 10)}T00:00`);
  if (nextDay) date.setDate(date.getDate() + 1);
  return date.toISOString();
};

// The backend stores commute and gathering in whole minutes.
const minuteSeconds = (value: FormDataEntryValue | null) => {
  const minutes = Math.round(Number(value || 0));
  return Number.isFinite(minutes) && minutes > 0
    ? minutes * SECONDS_PER_MINUTE
    : 0;
};

export const buildEventPayload = (form: FormData): EventPayload => {
  const isAllDay = form.get("is_all_day") === "true";
  const isAttendable = form.get("kind") === "attendable";
  // Gathering and commute only apply to timed attendable events.
  const allowsAttendance = isAttendable && !isAllDay;
  const gatheringSeconds = allowsAttendance
    ? minuteSeconds(form.get("gathering_minutes"))
    : 0;
  return {
    title: String(form.get("title") || ""),
    description: String(form.get("description") || "") || null,
    url: String(form.get("url") || "") || null,
    kind: isAttendable ? "attendable" : "informational",
    is_skipped: isAttendable && form.get("is_skipped") === "true",
    starts_at: isAllDay
      ? localDayStart(form.get("starts_at"))
      : localDateTime(form.get("starts_at")),
    ends_at: isAllDay
      ? localDayStart(form.get("ends_at"), true)
      : localDateTime(form.get("ends_at")),
    is_all_day: isAllDay,
    start_is_approximate:
      !isAllDay && form.get("start_is_approximate") === "true",
    end_is_approximate: !isAllDay && form.get("end_is_approximate") === "true",
    location: String(form.get("location") || "") || null,
    gathering_seconds: gatheringSeconds,
    gathering_is_optional:
      gatheringSeconds > 0 && form.get("gathering_is_optional") === "true",
    gathering_label:
      (gatheringSeconds > 0 &&
        String(form.get("gathering_label") || "").trim()) ||
      null,
    commute_before_seconds: allowsAttendance
      ? minuteSeconds(form.get("commute_before_minutes"))
      : 0,
    commute_after_seconds: allowsAttendance
      ? minuteSeconds(form.get("commute_after_minutes"))
      : 0,
  };
};

const buildTaskPayload = (form: FormData): TaskPayload => ({
  title: String(form.get("title") || ""),
  description: String(form.get("description") || "") || null,
  url: String(form.get("url") || "") || null,
  deadline_at: localDateTime(form.get("deadline_at")),
});

export const buildTimelineItemRequest = (
  type: "event" | "task",
  editingItemId: string | null,
  form: FormData,
  mainTagId: string | null,
): TimelineItemRequest => {
  const resource = type === "event" ? "events" : "tasks";
  const body =
    type === "event"
      ? { ...buildEventPayload(form), main_tag_id: mainTagId }
      : buildTaskPayload(form);

  return {
    path: `/api/${resource}${editingItemId ? `/${editingItemId}` : ""}`,
    method: editingItemId ? "PUT" : "POST",
    body,
  };
};
