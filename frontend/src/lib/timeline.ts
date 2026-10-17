import {
  MINUTES_PER_HOUR,
  SECONDS_PER_MINUTE,
  dateKey,
  gridOffset,
  localDateTime,
  minutesIntoDay,
} from "./calendar";
import type { Tag, TimelineEntry } from "../types";

export type EventPayload = {
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
    height: gridOffset(currentMinutes),
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

// Whether an entry takes up the user's time. Heads-up (informational) events
// and skipped events stay visible but never push busy entries aside.
export const isBusy = (entry: TimelineEntry) =>
  entry.type !== "event" || (entry.kind === "attendable" && !entry.is_skipped);

const isTimedAttendable = (entry: TimelineEntry) =>
  entry.type === "event" &&
  entry.kind === "attendable" &&
  !entry.is_all_day &&
  !entry.is_skipped;

// Commute and gathering only apply to timed attendable events the user is
// going to; ignore them on anything else. A skipped event keeps its stored
// values for when the user changes their mind.
export const getCommuteSeconds = (entry: TimelineEntry) =>
  isTimedAttendable(entry)
    ? {
        before: entry.commute_before_seconds ?? 0,
        after: entry.commute_after_seconds ?? 0,
      }
    : { before: 0, after: 0 };

export const getGathering = (entry: TimelineEntry) => {
  const seconds = isTimedAttendable(entry) ? (entry.gathering_seconds ?? 0) : 0;
  return {
    seconds,
    isOptional: seconds > 0 && Boolean(entry.gathering_is_optional),
    label: seconds > 0 ? entry.gathering_label : undefined,
  };
};

const secondsBeforeStart = (entry: TimelineEntry, seconds: number) =>
  new Date(new Date(entry.starts_at ?? 0).getTime() - seconds * 1000);

// When people gather; starts_at stays the real start of the event.
export const getGatheringStart = (entry: TimelineEntry) =>
  secondsBeforeStart(entry, getGathering(entry).seconds);

// Where the event block begins. Mandatory gathering is part of the block;
// optional gathering is a separate band above it that may be skipped.
export const getBodyStart = (entry: TimelineEntry) => {
  const gathering = getGathering(entry);
  return secondsBeforeStart(
    entry,
    gathering.isOptional ? 0 : gathering.seconds,
  );
};

// The top of everything drawn for an event: commute, then any gathering.
export const getDrawnStart = (entry: TimelineEntry) =>
  new Date(
    getGatheringStart(entry).getTime() - getCommuteSeconds(entry).before * 1000,
  );

const occupiedRanges = new WeakMap<TimelineEntry, { start: Date; end: Date }>();

// The span an event blocks on the calendar: mandatory gathering and commute.
// Optional gathering only counts when there is a commute to it, because then
// the user has planned to be there. Cached per entry object because overlap
// checks compare every pair in a day.
export const getOccupiedRange = (entry: TimelineEntry) => {
  const cached = occupiedRanges.get(entry);
  if (cached) return cached;

  const { before, after } = getCommuteSeconds(entry);
  const end = new Date(entry.ends_at ?? 0);
  const range = {
    start: before ? getDrawnStart(entry) : getBodyStart(entry),
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
