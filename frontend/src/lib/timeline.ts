import { localDateTime, minutesIntoDay } from "./calendar";
import type { Tag, TimelineEntry } from "../types";

export type EventPayload = {
  title: string;
  description: string | null;
  url: string | null;
  kind: FormDataEntryValue | null;
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

export const buildEventPayload = (form: FormData): EventPayload => ({
  title: String(form.get("title") || ""),
  description: String(form.get("description") || "") || null,
  url: String(form.get("url") || "") || null,
  kind: form.get("kind") || "informational",
  starts_at: localDateTime(form.get("starts_at")),
  ends_at: localDateTime(form.get("ends_at")),
  location: form.get("location") || null,
});

export const buildTaskPayload = (form: FormData): TaskPayload => ({
  title: String(form.get("title") || ""),
  description: String(form.get("description") || "") || null,
  url: String(form.get("url") || "") || null,
  deadline_at: localDateTime(form.get("deadline_at")),
});
