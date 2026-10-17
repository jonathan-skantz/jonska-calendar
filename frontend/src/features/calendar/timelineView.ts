import { addDays, addMinutes, startOfDay } from "date-fns";
import type { MouseEvent } from "react";
import {
  DEFAULT_EVENT_DURATION_MINUTES,
  HOURS_PER_DAY,
  LAST_EVENT_START_HOUR,
  MINIMUM_ENTRY_DURATION_MINUTES,
  MINUTES_PER_HOUR,
  SECONDS_PER_MINUTE,
  TASK_DISPLAY_DURATION_MINUTES,
  WEEK_DAY_COUNT,
  dateKey,
  formatTime,
  gridOffset,
  localDateKey,
  minutesIntoDay,
} from "../../lib/calendar";
import {
  getBodyStart,
  getCommuteSeconds,
  getDrawnStart,
  getGathering,
  getGatheringStart,
  getElapsedOverlayState,
  getEntryMinutes,
  getOccupiedRange,
  getTimelineEntryDate,
  isEventVisible,
} from "../../lib/timeline";
import type { TimelineEntry } from "../../types";
import type { TimeFormatPreference } from "../../lib/calendar";

export type EventPreview = {
  top: string;
  height: string;
  startsAt: string;
  endsAt: string;
};

export type CommuteSegment = {
  position: "before" | "after";
  minutes: number;
  startMinutes: number;
  durationMinutes: number;
};

export type GatheringSegment = {
  minutes: number;
  isOptional: boolean;
  label?: string;
  startMinutes: number;
  durationMinutes: number;
  // True when gathering began on an earlier day than this segment.
  continuesFromPreviousDay: boolean;
  // False when the event itself starts on a later day than this segment.
  endsAtEventStart: boolean;
};

export type AllDayEventSpan = {
  entry: TimelineEntry;
  startIndex: number;
  endIndex: number;
  lane: number;
  continuesFromPreviousWeek: boolean;
  continuesToNextWeek: boolean;
};

export const getCalendarDates = (
  rangeStart: Date,
  dayCount: number = WEEK_DAY_COUNT,
) => Array.from({ length: dayCount }, (_, index) => addDays(rangeStart, index));

const eventOverlapsRange = (
  entry: TimelineEntry,
  rangeStart: Date,
  rangeEnd: Date,
) => {
  if (entry.type !== "event" || !entry.starts_at || !entry.ends_at) {
    return false;
  }

  // Drawn rather than occupied start, so an optional gathering band that
  // crosses midnight still shows on the previous day.
  return (
    getDrawnStart(entry) < rangeEnd && getOccupiedRange(entry).end > rangeStart
  );
};

export const getVisibleWeekEntries = (
  entries: TimelineEntry[],
  dates: Date[],
  hiddenTagIds: string[],
) => {
  if (dates.length === 0) return [];

  const rangeStart = startOfDay(dates[0]);
  const rangeEnd = addDays(startOfDay(dates[dates.length - 1]), 1);
  return entries.filter((entry) => {
    const isInWeek =
      entry.type === "event"
        ? eventOverlapsRange(entry, rangeStart, rangeEnd)
        : dates.some(
            (date) =>
              dateKey(date) === localDateKey(getTimelineEntryDate(entry)),
          );
    return isInWeek && isEventVisible(entry, hiddenTagIds);
  });
};

const layoutStartTime = (entry: TimelineEntry) =>
  entry.type === "event"
    ? getOccupiedRange(entry).start.getTime()
    : new Date(getTimelineEntryDate(entry) ?? 0).getTime();

// Ordered by where each entry visually starts, commute included, because
// overlap depth and adjacency compare each entry with its neighbours.
export const getEntriesForDate = (entries: TimelineEntry[], date: Date) => {
  const rangeStart = startOfDay(date);
  const rangeEnd = addDays(rangeStart, 1);
  return entries
    .filter((entry) =>
      entry.type === "event"
        ? eventOverlapsRange(entry, rangeStart, rangeEnd)
        : localDateKey(getTimelineEntryDate(entry)) === dateKey(date),
    )
    .sort((left, right) => layoutStartTime(left) - layoutStartTime(right));
};

export const getAllDayEventSpans = (
  entries: TimelineEntry[],
  dates: Date[],
): AllDayEventSpan[] => {
  if (!dates.length) return [];

  const weekStart = dateKey(dates[0]);
  const weekEnd = dateKey(addDays(dates[dates.length - 1], 1));
  const spans = entries
    .filter(
      (entry) =>
        entry.type === "event" &&
        entry.is_all_day &&
        entry.starts_at &&
        entry.ends_at,
    )
    .map((entry) => {
      const startsOn = localDateKey(entry.starts_at);
      const endsBefore = localDateKey(entry.ends_at);
      const firstDay = dates.findIndex((date) => dateKey(date) >= startsOn);
      const afterLastDay = dates.findIndex(
        (date) => dateKey(date) >= endsBefore,
      );
      return {
        entry,
        startIndex: firstDay < 0 ? dates.length - 1 : firstDay,
        endIndex: afterLastDay < 0 ? dates.length - 1 : afterLastDay - 1,
        continuesFromPreviousWeek: startsOn < weekStart,
        continuesToNextWeek: endsBefore > weekEnd,
      };
    })
    .filter((span) => span.startIndex <= span.endIndex)
    .sort(
      (left, right) =>
        left.startIndex - right.startIndex || right.endIndex - left.endIndex,
    );

  const laneSpans: { startIndex: number; endIndex: number }[][] = [];
  return spans.map((span) => {
    let lane = laneSpans.findIndex((items) =>
      items.every(
        (item) =>
          span.endIndex < item.startIndex || span.startIndex > item.endIndex,
      ),
    );
    if (lane < 0) {
      lane = laneSpans.length;
      laneSpans.push([]);
    }
    laneSpans[lane].push(span);
    return { ...span, lane };
  });
};

export const getClickedEventStart = (
  event: MouseEvent<HTMLDivElement>,
  date: Date,
) => {
  const bounds = event.currentTarget.getBoundingClientRect();
  const hour = Math.max(
    0,
    Math.min(
      LAST_EVENT_START_HOUR,
      Math.round(
        (event.clientY - bounds.top) / (bounds.height / HOURS_PER_DAY),
      ),
    ),
  );
  const start = new Date(date);
  start.setHours(hour, 0, 0, 0);
  return start;
};

export const getEventPreview = (
  startAt: string | null,
  date: Date,
  timeFormat: TimeFormatPreference = "browser",
): EventPreview | null => {
  if (!startAt || localDateKey(startAt) !== dateKey(date)) return null;
  const end = addMinutes(new Date(startAt), DEFAULT_EVENT_DURATION_MINUTES);
  return {
    top: gridOffset(minutesIntoDay(startAt)),
    height: gridOffset(DEFAULT_EVENT_DURATION_MINUTES),
    startsAt: formatTime(startAt, { preference: timeFormat }),
    endsAt: formatTime(end, { preference: timeFormat }),
  };
};

export const getDayElapsedOverlay = (
  date: Date,
  today: Date,
  formatLabel: (value: Date) => string,
) => getElapsedOverlayState(date, today, formatLabel);

const occupiedEdgesTouch = (earlier: TimelineEntry, later: TimelineEntry) =>
  getOccupiedRange(earlier).end.getTime() ===
  getOccupiedRange(later).start.getTime();

export const getEventAdjacency = (
  entry: TimelineEntry,
  previous?: TimelineEntry,
  next?: TimelineEntry,
) => ({
  trimTop:
    entry.type === "event" &&
    previous?.type === "event" &&
    occupiedEdgesTouch(previous, entry),
  trimBottom:
    entry.type === "event" &&
    next?.type === "event" &&
    occupiedEdgesTouch(entry, next),
});

export const entriesOverlap = (left: TimelineEntry, right: TimelineEntry) => {
  if (left.type !== "event" || right.type !== "event") return false;
  const leftRange = getOccupiedRange(left);
  const rightRange = getOccupiedRange(right);
  return leftRange.start < rightRange.end && rightRange.start < leftRange.end;
};

export const getEventOverlapDepth = (
  entry: TimelineEntry,
  earlierEntries: TimelineEntry[],
) => {
  if (entry.type !== "event" || !entry.starts_at) return 0;

  const startsAt = getOccupiedRange(entry).start;
  return earlierEntries.filter(
    (earlierEntry) =>
      earlierEntry.type === "event" &&
      earlierEntry.ends_at &&
      getOccupiedRange(earlierEntry).end > startsAt,
  ).length;
};

const clipToDay = (start: Date, end: Date, date: Date) => {
  const dayStart = startOfDay(date);
  const dayEnd = addDays(dayStart, 1);
  const segmentStart = start > dayStart ? start : dayStart;
  const segmentEnd = end < dayEnd ? end : dayEnd;
  if (segmentStart >= segmentEnd) return null;

  const segmentStartMinutes = minutesIntoDay(segmentStart);
  const segmentEndMinutes =
    segmentEnd.getTime() === dayEnd.getTime()
      ? HOURS_PER_DAY * MINUTES_PER_HOUR
      : minutesIntoDay(segmentEnd);
  return {
    startMinutes: segmentStartMinutes,
    durationMinutes: segmentEndMinutes - segmentStartMinutes,
    continuesFromPreviousDay: start < dayStart,
    continuesToNextDay: end > dayEnd,
  };
};

export const eventBodyOverlapsDate = (entry: TimelineEntry, date: Date) => {
  if (entry.type !== "event") return true;
  if (!entry.starts_at || !entry.ends_at) return false;
  const dayStart = startOfDay(date);
  return (
    getBodyStart(entry) < addDays(dayStart, 1) &&
    new Date(entry.ends_at) > dayStart
  );
};

export const getCommuteSegments = (
  entry: TimelineEntry,
  date: Date,
): CommuteSegment[] => {
  if (entry.type !== "event" || !entry.starts_at || !entry.ends_at) return [];

  const endsAt = new Date(entry.ends_at);
  const { end } = getOccupiedRange(entry);
  const commute = getCommuteSeconds(entry);
  const segments: CommuteSegment[] = [];
  // Commute ends where the first drawn piece begins, gathering included.
  const before = clipToDay(
    getDrawnStart(entry),
    getGatheringStart(entry),
    date,
  );
  if (before && commute.before) {
    segments.push({
      position: "before",
      minutes: Math.round(commute.before / SECONDS_PER_MINUTE),
      startMinutes: before.startMinutes,
      durationMinutes: before.durationMinutes,
    });
  }
  const after = clipToDay(endsAt, end, date);
  if (after && commute.after) {
    segments.push({
      position: "after",
      minutes: Math.round(commute.after / SECONDS_PER_MINUTE),
      startMinutes: after.startMinutes,
      durationMinutes: after.durationMinutes,
    });
  }
  return segments;
};

export const getGatheringSegment = (
  entry: TimelineEntry,
  date: Date,
): GatheringSegment | null => {
  const gathering = getGathering(entry);
  if (!gathering.seconds || !entry.starts_at) return null;

  const segment = clipToDay(
    getGatheringStart(entry),
    new Date(entry.starts_at),
    date,
  );
  return segment
    ? {
        minutes: Math.round(gathering.seconds / SECONDS_PER_MINUTE),
        isOptional: gathering.isOptional,
        label: gathering.label,
        startMinutes: segment.startMinutes,
        durationMinutes: segment.durationMinutes,
        continuesFromPreviousDay: segment.continuesFromPreviousDay,
        endsAtEventStart: !segment.continuesToNextDay,
      }
    : null;
};

export const getCalendarEntryLayout = (
  entry: TimelineEntry,
  now: Date,
  date = new Date(getTimelineEntryDate(entry) ?? now),
) => {
  const startMinutes = getEntryMinutes(entry);
  // An event is only done once the commute home is done too.
  const fullyElapsed =
    entry.type === "event" &&
    Boolean(entry.ends_at && getOccupiedRange(entry).end <= now);

  if (entry.type === "event") {
    if (!entry.starts_at || !entry.ends_at) {
      return {
        startMinutes,
        durationMinutes: MINIMUM_ENTRY_DURATION_MINUTES,
        fullyElapsed,
        continuesFromPreviousDay: false,
        continuesToNextDay: false,
      };
    }

    const segment = clipToDay(
      getBodyStart(entry),
      new Date(entry.ends_at),
      date,
    );
    if (segment) return { ...segment, fullyElapsed };

    return {
      startMinutes,
      durationMinutes: MINIMUM_ENTRY_DURATION_MINUTES,
      fullyElapsed,
      continuesFromPreviousDay: false,
      continuesToNextDay: false,
    };
  }

  return {
    startMinutes,
    durationMinutes: Math.max(
      TASK_DISPLAY_DURATION_MINUTES,
      MINIMUM_ENTRY_DURATION_MINUTES,
    ),
    fullyElapsed: false,
    continuesFromPreviousDay: false,
    continuesToNextDay: false,
  };
};
