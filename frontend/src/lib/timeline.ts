import { localDateTime, minutesIntoDay } from "./calendar";
import type { TimelineEntry } from "../types";

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
