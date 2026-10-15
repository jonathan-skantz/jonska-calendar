import {
  CALENDAR_GRID_SCALE,
  MINUTES_PER_HOUR,
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
  location: FormDataEntryValue | null;
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
  body: EventPayload | TaskPayload | (EventPayload & { main_tag_id: string | null });
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

  const currentMinutes = today.getHours() * MINUTES_PER_HOUR + today.getMinutes();
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

export const buildEventPayload = (form: FormData): EventPayload => ({
  title: String(form.get("title") || ""),
  description: String(form.get("description") || "") || null,
  url: String(form.get("url") || "") || null,
  kind: form.get("kind") === "attendable" ? "attendable" : "informational",
  starts_at: localDateTime(form.get("starts_at")),
  ends_at: localDateTime(form.get("ends_at")),
  location: String(form.get("location") || "") || null,
});

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
