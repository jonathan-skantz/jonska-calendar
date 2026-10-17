import { minutesIntoDay } from "./calendar";
import type { Tag, TimelineEntry } from "../types";

type EventTagSelection = {
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
