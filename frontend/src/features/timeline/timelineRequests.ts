import {
  SECONDS_PER_MINUTE,
  browserTimeZone,
  localDateTime,
} from "../../lib/calendar";
import { readRecurrence } from "../../lib/recurrence";
import type { Recurrence, Tag, TimelineEntry } from "../../types";

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
  // The times repeat in the browser's time zone.
  recurrence: Recurrence | null;
  time_zone: string;
};

// Which occurrences of a repeating event a change applies to.
export type OccurrenceScope = "this" | "following" | "all";

// The path of a change to an event, or to some occurrences of it.
const eventPath = (entry: TimelineEntry, scope: OccurrenceScope) =>
  entry.occurrence_start
    ? `/api/events/${entry.id}?${new URLSearchParams({
        scope,
        occurrence: entry.occurrence_start,
      })}`
    : `/api/events/${entry.id}`;

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
    recurrence: readRecurrence(form),
    time_zone: browserTimeZone(),
  };
};

const buildTaskPayload = (form: FormData): TaskPayload => ({
  title: String(form.get("title") || ""),
  description: String(form.get("description") || "") || null,
  url: String(form.get("url") || "") || null,
  deadline_at: localDateTime(form.get("deadline_at")),
});

const sameText = (a?: string | null, b?: string | null) =>
  (a?.trim() || null) === (b?.trim() || null);

const sameRecurrence = (a?: Recurrence | null, b?: Recurrence | null) =>
  a?.frequency === b?.frequency &&
  (a?.interval ?? 1) === (b?.interval ?? 1) &&
  (a?.weekdays ?? []).join() === (b?.weekdays ?? []).join() &&
  (a?.monthly_by ?? "") === (b?.monthly_by ?? "") &&
  (a?.until ?? "") === (b?.until ?? "") &&
  (a?.count ?? 0) === (b?.count ?? 0);

// Whether the form changes an occurrence only in ways one occurrence can
// differ from its series: its time and whether it is skipped.
export const changesOnlyOccurrence = (
  occurrence: TimelineEntry,
  form: FormData,
  selectedTagIds: string[],
  mainTagId: string | null,
) => {
  const payload = buildEventPayload(form);
  const tagIds = (occurrence.tags ?? []).map((tag) => tag.id);
  return (
    payload.title.trim() === occurrence.title &&
    sameText(payload.description, occurrence.description) &&
    sameText(payload.url, occurrence.url) &&
    sameText(payload.location as string | null, occurrence.location) &&
    payload.kind === occurrence.kind &&
    payload.is_all_day === Boolean(occurrence.is_all_day) &&
    payload.start_is_approximate === Boolean(occurrence.start_is_approximate) &&
    payload.end_is_approximate === Boolean(occurrence.end_is_approximate) &&
    payload.gathering_seconds === (occurrence.gathering_seconds ?? 0) &&
    payload.gathering_is_optional ===
      Boolean(occurrence.gathering_is_optional) &&
    sameText(payload.gathering_label, occurrence.gathering_label) &&
    payload.commute_before_seconds ===
      (occurrence.commute_before_seconds ?? 0) &&
    payload.commute_after_seconds === (occurrence.commute_after_seconds ?? 0) &&
    sameRecurrence(payload.recurrence, occurrence.recurrence) &&
    mainTagId === (occurrence.main_tag_id ?? null) &&
    selectedTagIds.length === tagIds.length &&
    selectedTagIds.every((id) => tagIds.includes(id))
  );
};

// A saved occurrence of a repeating event changes the occurrences scope
// names; scope is ignored otherwise.
export const buildTimelineItemRequest = (
  type: "event" | "task",
  editingItem: TimelineEntry | null,
  form: FormData,
  mainTagId: string | null,
  scope: OccurrenceScope = "all",
): TimelineItemRequest => {
  if (type === "task") {
    return {
      path: editingItem ? `/api/tasks/${editingItem.id}` : "/api/tasks",
      method: editingItem ? "PUT" : "POST",
      body: buildTaskPayload(form),
    };
  }
  return {
    path: editingItem ? eventPath(editingItem, scope) : "/api/events",
    method: editingItem ? "PUT" : "POST",
    body: { ...buildEventPayload(form), main_tag_id: mainTagId },
  };
};

// Saves an item moved in the week view: its own fields with the new times.
// Leaving out main_tag_id keeps the event's main tag. A moved occurrence
// moves alone unless scope says otherwise.
export const buildMoveRequest = (
  moved: TimelineEntry,
  scope: OccurrenceScope = "this",
): { path: string; body: EventPayload | TaskPayload } =>
  moved.type === "task"
    ? {
        path: `/api/tasks/${moved.id}`,
        body: {
          title: moved.title,
          description: moved.description ?? null,
          url: moved.url ?? null,
          deadline_at: moved.deadline_at,
        },
      }
    : {
        path: eventPath(moved, scope),
        body: {
          title: moved.title,
          description: moved.description ?? null,
          url: moved.url ?? null,
          kind: moved.kind ?? "attendable",
          is_skipped: Boolean(moved.is_skipped),
          starts_at: moved.starts_at,
          ends_at: moved.ends_at,
          is_all_day: Boolean(moved.is_all_day),
          start_is_approximate: Boolean(moved.start_is_approximate),
          end_is_approximate: Boolean(moved.end_is_approximate),
          location: moved.location ?? null,
          gathering_seconds: moved.gathering_seconds ?? 0,
          gathering_is_optional: Boolean(moved.gathering_is_optional),
          gathering_label: moved.gathering_label ?? null,
          commute_before_seconds: moved.commute_before_seconds ?? 0,
          commute_after_seconds: moved.commute_after_seconds ?? 0,
          recurrence: moved.recurrence ?? null,
          time_zone: browserTimeZone(),
        },
      };
