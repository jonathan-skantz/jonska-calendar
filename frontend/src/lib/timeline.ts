import {
  CALENDAR_GRID_SCALE,
  MINUTES_PER_HOUR,
  SECONDS_PER_MINUTE,
  dateKey,
  localDateTime,
  minutesIntoDay,
} from "./calendar";
import type { Tag, TimelineEntry } from "../types";

export type EventPayload = {
  title: string;
  description: string | null;
  url: string | null;
  kind: "attendable" | "informational";
  starts_at: string | undefined;
  ends_at: string | undefined;
  is_all_day: boolean;
  start_is_approximate: boolean;
  end_is_approximate: boolean;
  location: FormDataEntryValue | null;
  commute_before_seconds: number;
  commute_after_seconds: number;
};

export type TaskPayload = {
  title: string;
  description: string | null;
  url: string | null;
  deadline_at: string | undefined;
};

export type EventTagSelection = {
  tagIds: string[];
  mainTagId: string | null;
};

export const getDefaultEventTagSelection = (tags: Tag[]): EventTagSelection => {
  const defaultTag = tags.find((tag) => tag.is_default);
  return {
    tagIds: defaultTag ? [defaultTag.id] : [],
    mainTagId: defaultTag?.id ?? tags[0]?.id ?? null,
  };
};

export const getEventTagChanges = (previous: Tag[], nextTagIds: string[]) => {
  const previousTagIds = new Set(previous.map((tag) => tag.id));
  const nextIDs = new Set(nextTagIds);
  return {
    add: nextTagIds.filter((tagId) => !previousTagIds.has(tagId)),
    remove: previous.filter((tag) => !nextIDs.has(tag.id)),
  };
};

export type TimelineItemRequest = {
  path: string;
  method: "POST" | "PUT";
  body:
    | EventPayload
    | TaskPayload
    | (EventPayload & { main_tag_id: string | null });
};

export type ElapsedOverlayState =
  | { kind: "past" }
  | { kind: "today"; height: string; label: string }
  | { kind: "none" };

export const getElapsedOverlayState = (
  date: Date,
  today: Date,
  formatTimeLabel: (value: Date) => string,
): ElapsedOverlayState => {
  const dayKey = dateKey(date);
  const todayKey = dateKey(today);
  if (dayKey < todayKey) return { kind: "past" };
  if (dayKey !== todayKey) return { kind: "none" };

  const currentMinutes =
    today.getHours() * MINUTES_PER_HOUR + today.getMinutes();
  if (currentMinutes === 0) return { kind: "none" };
  return {
    kind: "today",
    height: `${(currentMinutes / MINUTES_PER_HOUR) * CALENDAR_GRID_SCALE}rem`,
    label: formatTimeLabel(today),
  };
};

export const getTimelineEntryDate = (entry: TimelineEntry) =>
  entry.type === "event" ? entry.starts_at : entry.deadline_at;

export const getEntryMinutes = (entry: TimelineEntry) =>
  minutesIntoDay(getTimelineEntryDate(entry));

export const getMainTag = (entry: TimelineEntry) =>
  entry.tags?.find((tag) => tag.id === entry.main_tag_id) ?? entry.tags?.[0];

export const getOrderedStripTags = (entry: TimelineEntry): Tag[] => {
  const mainTag = getMainTag(entry);
  const additionalTags =
    entry.type === "event"
      ? [...(entry.tags?.filter((tag) => tag.id !== mainTag?.id) ?? [])].sort(
          (left, right) => left.name.localeCompare(right.name),
        )
      : [];
  return mainTag ? [mainTag, ...additionalTags] : additionalTags;
};

export const isEventVisible = (entry: TimelineEntry, hiddenTagIds: string[]) =>
  entry.type !== "event" ||
  !entry.tags?.some((tag) => hiddenTagIds.includes(tag.id));

const localDayStart = (value: FormDataEntryValue | null, nextDay = false) => {
  if (typeof value !== "string" || !value) return undefined;
  const date = new Date(`${value.slice(0, 10)}T00:00`);
  if (nextDay) date.setDate(date.getDate() + 1);
  return date.toISOString();
};

// The backend stores commute in whole minutes.
const commuteSeconds = (value: FormDataEntryValue | null) => {
  const minutes = Math.round(Number(value || 0));
  return Number.isFinite(minutes) && minutes > 0
    ? minutes * SECONDS_PER_MINUTE
    : 0;
};

export const buildEventPayload = (form: FormData): EventPayload => {
  const isAllDay = form.get("is_all_day") === "true";
  const isAttendable = form.get("kind") === "attendable";
  const allowsCommute = isAttendable && !isAllDay;
  return {
    title: String(form.get("title") || ""),
    description: String(form.get("description") || "") || null,
    url: String(form.get("url") || "") || null,
    kind: isAttendable ? "attendable" : "informational",
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
    commute_before_seconds: allowsCommute
      ? commuteSeconds(form.get("commute_before_minutes"))
      : 0,
    commute_after_seconds: allowsCommute
      ? commuteSeconds(form.get("commute_after_minutes"))
      : 0,
  };
};

// Commute only applies to timed attendable events; ignore it on anything else.
export const getCommuteSeconds = (entry: TimelineEntry) =>
  entry.type === "event" && entry.kind === "attendable" && !entry.is_all_day
    ? {
        before: entry.commute_before_seconds ?? 0,
        after: entry.commute_after_seconds ?? 0,
      }
    : { before: 0, after: 0 };

const occupiedRanges = new WeakMap<TimelineEntry, { start: Date; end: Date }>();

// The span an event blocks on the calendar, including commute on both sides.
// Cached per entry object because overlap checks compare every pair in a day.
export const getOccupiedRange = (entry: TimelineEntry) => {
  const cached = occupiedRanges.get(entry);
  if (cached) return cached;

  const { before, after } = getCommuteSeconds(entry);
  const start = new Date(entry.starts_at ?? 0);
  const end = new Date(entry.ends_at ?? 0);
  const range = {
    start: new Date(start.getTime() - before * 1000),
    end: new Date(end.getTime() + after * 1000),
  };
  occupiedRanges.set(entry, range);
  return range;
};

export const buildTaskPayload = (form: FormData): TaskPayload => ({
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
