import { addDays, startOfDay } from "date-fns";
import type { MouseEvent } from "react";
import {
  CALENDAR_GRID_SCALE,
  DEFAULT_EVENT_DURATION_MINUTES,
  HOURS_PER_DAY,
  LAST_EVENT_START_HOUR,
  MINIMUM_ENTRY_DURATION_MINUTES,
  MINUTES_PER_HOUR,
  TASK_DISPLAY_DURATION_MINUTES,
  WEEK_DAY_COUNT,
  dateKey,
  formatTime,
  localDateKey,
  minutesIntoDay,
  sameInstant,
} from "../../lib/calendar";
import {
  getElapsedOverlayState,
  getEntryMinutes,
  getTimelineEntryDate,
  isEventVisible,
} from "../../lib/timeline";
import type { TimelineEntry } from "../../types";
import type { TimeFormatPreference } from "../../lib/calendar";

export type EventPreview = {
  top: string;
  startsAt: string;
  endsAt: string;
};

export const getCalendarDates = (weekStart: Date) =>
  Array.from({ length: WEEK_DAY_COUNT }, (_, index) =>
    addDays(weekStart, index),
  );

const eventOverlapsRange = (
  entry: TimelineEntry,
  rangeStart: Date,
  rangeEnd: Date,
) => {
  if (entry.type !== "event" || !entry.starts_at || !entry.ends_at) {
    return false;
  }

  const startsAt = new Date(entry.starts_at);
  const endsAt = new Date(entry.ends_at);
  return startsAt < rangeEnd && endsAt > rangeStart;
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

export const getEntriesForDate = (entries: TimelineEntry[], date: Date) => {
  const rangeStart = startOfDay(date);
  const rangeEnd = addDays(rangeStart, 1);
  return entries.filter((entry) =>
    entry.type === "event"
      ? eventOverlapsRange(entry, rangeStart, rangeEnd)
      : localDateKey(getTimelineEntryDate(entry)) === dateKey(date),
  );
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
  const start = new Date(startAt);
  const end = new Date(
    start.getTime() +
      DEFAULT_EVENT_DURATION_MINUTES * MINUTES_PER_HOUR * 60 * 1000,
  ).toISOString();
  return {
    top: `${(minutesIntoDay(startAt) / MINUTES_PER_HOUR) * CALENDAR_GRID_SCALE}rem`,
    startsAt: formatTime(startAt, { preference: timeFormat }),
    endsAt: formatTime(end, { preference: timeFormat }),
  };
};

export const getDayElapsedOverlay = (
  date: Date,
  today: Date,
  formatLabel: (value: Date) => string,
) => getElapsedOverlayState(date, today, formatLabel);

export const getEventAdjacency = (
  entry: TimelineEntry,
  previous?: TimelineEntry,
  next?: TimelineEntry,
) => ({
  trimTop:
    entry.type === "event" &&
    previous?.type === "event" &&
    sameInstant(previous.ends_at, entry.starts_at),
  trimBottom:
    entry.type === "event" &&
    next?.type === "event" &&
    sameInstant(entry.ends_at, next.starts_at),
});

export const getEventOverlapDepth = (
  entry: TimelineEntry,
  earlierEntries: TimelineEntry[],
) => {
  if (entry.type !== "event" || !entry.starts_at) return 0;

  const startsAt = new Date(entry.starts_at).getTime();
  return earlierEntries.filter(
    (earlierEntry) =>
      earlierEntry.type === "event" &&
      earlierEntry.ends_at &&
      new Date(earlierEntry.ends_at).getTime() > startsAt,
  ).length;
};

export const getCalendarEntryLayout = (
  entry: TimelineEntry,
  now: Date,
  date = new Date(getTimelineEntryDate(entry) ?? now),
) => {
  const startMinutes = getEntryMinutes(entry);
  const fullyElapsed =
    entry.type === "event" &&
    Boolean(
      entry.ends_at && new Date(entry.ends_at).getTime() <= now.getTime(),
    );

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

    const dayStart = startOfDay(date);
    const dayEnd = addDays(dayStart, 1);
    const eventStart = new Date(entry.starts_at);
    const eventEnd = new Date(entry.ends_at);
    const segmentStart = eventStart > dayStart ? eventStart : dayStart;
    const segmentEnd = eventEnd < dayEnd ? eventEnd : dayEnd;

    if (segmentStart < segmentEnd) {
      const segmentStartMinutes =
        segmentStart.getTime() === dayStart.getTime()
          ? 0
          : segmentStart.getHours() * MINUTES_PER_HOUR +
            segmentStart.getMinutes();
      const segmentEndMinutes =
        segmentEnd.getTime() === dayEnd.getTime()
          ? HOURS_PER_DAY * MINUTES_PER_HOUR
          : segmentEnd.getHours() * MINUTES_PER_HOUR + segmentEnd.getMinutes();
      return {
        startMinutes: segmentStartMinutes,
        durationMinutes: segmentEndMinutes - segmentStartMinutes,
        fullyElapsed,
        continuesFromPreviousDay: eventStart < dayStart,
        continuesToNextDay: eventEnd > dayEnd,
      };
    }

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
