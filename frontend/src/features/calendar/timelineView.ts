import { addDays } from "date-fns";
import type { MouseEvent } from "react";
import {
  CALENDAR_GRID_SCALE,
  HOURS_PER_DAY,
  LAST_EVENT_START_HOUR,
  MINIMUM_ENTRY_DURATION_MINUTES,
  MINUTES_PER_HOUR,
  PREVIEW_EVENT_DURATION_MINUTES,
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

export type EventPreview = {
  top: string;
  startsAt: string;
  endsAt: string;
};

export const getCalendarDates = (weekStart: Date) =>
  Array.from({ length: WEEK_DAY_COUNT }, (_, index) =>
    addDays(weekStart, index),
  );

export const getVisibleWeekEntries = (
  entries: TimelineEntry[],
  dates: Date[],
  hiddenTagIds: string[],
) =>
  entries.filter((entry) => {
    const entryDate = getTimelineEntryDate(entry);
    const isInWeek = dates.some(
      (date) => dateKey(date) === localDateKey(entryDate),
    );
    return isInWeek && isEventVisible(entry, hiddenTagIds);
  });

export const getEntriesForDate = (entries: TimelineEntry[], date: Date) =>
  entries.filter(
    (entry) => localDateKey(getTimelineEntryDate(entry)) === dateKey(date),
  );

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
): EventPreview | null => {
  if (!startAt || localDateKey(startAt) !== dateKey(date)) return null;
  const start = new Date(startAt);
  const end = new Date(
    start.getTime() + PREVIEW_EVENT_DURATION_MINUTES * MINUTES_PER_HOUR * 60 * 1000,
  ).toISOString();
  return {
    top: `${(minutesIntoDay(startAt) / MINUTES_PER_HOUR) * CALENDAR_GRID_SCALE}rem`,
    startsAt: formatTime(startAt),
    endsAt: formatTime(end),
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

export const getCalendarEntryLayout = (
  entry: TimelineEntry,
  now: Date,
) => {
  const startMinutes = getEntryMinutes(entry);
  const endMinutes =
    entry.type === "event"
      ? minutesIntoDay(entry.ends_at)
      : startMinutes + TASK_DISPLAY_DURATION_MINUTES;
  const durationMinutes = Math.max(
    entry.type === "event" && endMinutes > startMinutes
      ? endMinutes - startMinutes
      : TASK_DISPLAY_DURATION_MINUTES,
    MINIMUM_ENTRY_DURATION_MINUTES,
  );
  const fullyElapsed =
    entry.type === "event" &&
    Boolean(entry.ends_at && new Date(entry.ends_at).getTime() <= now.getTime());
  return { startMinutes, durationMinutes, fullyElapsed };
};
