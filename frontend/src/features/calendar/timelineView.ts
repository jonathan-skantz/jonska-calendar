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
  DEFAULT_SCROLL_HOUR,
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
  getEntryMinutes,
  getOccupiedRange,
  getTimelineEntryDate,
  isBusy,
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

type AllDayEventSpan = {
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
// horizontal placement and adjacency compare each entry with its neighbours.
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

// Overlapping events that start within this many minutes of each other sit
// side by side. A later one cascades over the earlier block instead, indented
// so the earlier event's text stays visible beside it. An hour fits the title
// and time lines even at the smallest hour height.
const SIDE_BY_SIDE_START_MINUTES = 60;
// Share of an overlapped block's width, from its left edge, that a later block
// leaves uncovered, so the earlier block's text and end stay visible.
const CASCADE_INDENT_FRACTION = 0.4;
const FRACTION_EPSILON = 1e-9;

export type HorizontalPlacement = {
  // Edges as fractions of the day column's width.
  left: number;
  right: number;
  // Drawn over an earlier block, or in a column beside an earlier one.
  raised: boolean;
};

type Span = [left: number, right: number];

const FULL_WIDTH: HorizontalPlacement = { left: 0, right: 1, raised: false };

// The widest stretch of the day column that leaves every strip uncovered. Of
// equally wide stretches the rightmost wins, so cascades step in from the left.
const widestFreeSpan = (strips: Span[]): Span => {
  let best: Span = [0, 0];
  let cursor = 0;
  for (const [left, right] of [
    ...[...strips].sort((a, b) => a[0] - b[0]),
    [1, 1],
  ]) {
    const width = left - cursor;
    if (
      width > FRACTION_EPSILON &&
      width >= best[1] - best[0] - FRACTION_EPSILON
    )
      best = [cursor, left];
    cursor = Math.max(cursor, right);
  }
  return best;
};

// One placement per entry of a start-ordered layer. Overlapping events that
// start close together form a group that shares a span in equal columns; an
// event reuses the first column that is free by the time it starts. A group
// that starts later takes the widest span that keeps the left part of every
// block it overlaps visible: over one full-width block that is a cascade, and
// over several columns it fits beside or inside them without hiding any.
export const getHorizontalPlacements = (
  layer: TimelineEntry[],
): HorizontalPlacement[] => {
  type Group = { start: Date; span: Span; columns: number[][] };
  const groups: Group[] = [];
  const columnByIndex = new Map<number, { group: Group; column: number }>();

  const placementOf = (index: number): HorizontalPlacement => {
    const placed = columnByIndex.get(index);
    if (!placed) return FULL_WIDTH;
    const { group, column } = placed;
    const [spanLeft, spanRight] = group.span;
    const width = (spanRight - spanLeft) / group.columns.length;
    return {
      left: spanLeft + column * width,
      right: spanLeft + (column + 1) * width,
      raised: spanLeft > 0 || column > 0,
    };
  };

  layer.forEach((entry, index) => {
    if (entry.type !== "event" || !entry.starts_at || !entry.ends_at) return;
    const start = getOccupiedRange(entry).start;
    const overlappingMembers = [...columnByIndex.keys()].filter((member) =>
      entriesOverlap(layer[member], entry),
    );
    const target = groups.findLast((group) =>
      overlappingMembers.some(
        (member) => columnByIndex.get(member)?.group === group,
      ),
    );
    const joinsTarget =
      target && start < addMinutes(target.start, SIDE_BY_SIDE_START_MINUTES);
    const group: Group = joinsTarget
      ? target
      : {
          start,
          span: widestFreeSpan(
            overlappingMembers.map((member) => {
              const { left, right } = placementOf(member);
              return [left, left + (right - left) * CASCADE_INDENT_FRACTION];
            }),
          ),
          columns: [],
        };
    if (!joinsTarget) groups.push(group);
    // Members of a column never overlap, so its last member ends latest.
    let column = group.columns.findIndex(
      (members) => !entriesOverlap(layer[members[members.length - 1]], entry),
    );
    if (column < 0) column = group.columns.push([]) - 1;
    group.columns[column].push(index);
    columnByIndex.set(index, { group, column });
  });

  return layer.map((_, index) => placementOf(index));
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

  // Tasks, and events without a span on this day, get a fixed-size block.
  const fixedBlock = {
    startMinutes,
    durationMinutes: MINIMUM_ENTRY_DURATION_MINUTES,
    fullyElapsed,
    continuesFromPreviousDay: false,
    continuesToNextDay: false,
  };
  if (entry.type === "task") {
    return {
      ...fixedBlock,
      durationMinutes: Math.max(
        TASK_DISPLAY_DURATION_MINUTES,
        MINIMUM_ENTRY_DURATION_MINUTES,
      ),
    };
  }
  const segment =
    entry.starts_at && entry.ends_at
      ? clipToDay(getBodyStart(entry), new Date(entry.ends_at), date)
      : null;
  return segment ? { ...segment, fullyElapsed } : fixedBlock;
};

// One timed entry placed in a day column. The fields match CalendarEntry's
// layout props.
export type DayEntryLayout = ReturnType<typeof getCalendarEntryLayout> & {
  entry: TimelineEntry;
  trimTop: boolean;
  trimBottom: boolean;
  placement: HorizontalPlacement;
  commuteSegments: CommuteSegment[];
  gatheringSegment: GatheringSegment | null;
  showBody: boolean;
  overlapsGhost: boolean;
};

// Busy and not-busy (heads-up, skipped) entries are laid out as separate
// layers, so ghosts never push busy blocks aside. All-day events are drawn in
// the header instead.
export const layoutDayEntries = (
  entries: TimelineEntry[],
  date: Date,
  now: Date,
) => {
  const timed = getEntriesForDate(entries, date).filter(
    (entry) => entry.type !== "event" || !entry.is_all_day,
  );
  const ghosts = timed.filter((entry) => !isBusy(entry));
  const busy = timed.filter(isBusy);
  const layoutLayer = (layer: TimelineEntry[]): DayEntryLayout[] => {
    const placements = getHorizontalPlacements(layer);
    return layer.map((entry, index) => ({
      ...getCalendarEntryLayout(entry, now, date),
      ...getEventAdjacency(entry, layer[index - 1], layer[index + 1]),
      entry,
      placement: placements[index],
      commuteSegments: getCommuteSegments(entry, date),
      gatheringSegment: getGatheringSegment(entry, date),
      showBody: eventBodyOverlapsDate(entry, date),
      overlapsGhost:
        isBusy(entry) && ghosts.some((ghost) => entriesOverlap(entry, ghost)),
    }));
  };
  return { ghosts: layoutLayer(ghosts), busy: layoutLayer(busy) };
};

// The hour the grid scrolls to when it shows a range. With today in range, the
// previous hour stays visible so the current time isn't at the very top.
export const getScrollToHour = (dates: Date[], now: Date) =>
  dates.some((date) => dateKey(date) === dateKey(now))
    ? Math.max(DEFAULT_SCROLL_HOUR, now.getHours() - 1)
    : DEFAULT_SCROLL_HOUR;
