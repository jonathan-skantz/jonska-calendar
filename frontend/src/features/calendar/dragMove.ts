import {
  addMilliseconds,
  differenceInMilliseconds,
  roundToNearestMinutes,
  startOfDay,
} from "date-fns";
import { HOURS_PER_DAY, MINUTES_PER_HOUR } from "../../lib/calendar";
import { getTimelineEntryDate } from "../../lib/timeline";
import type { TimelineEntry } from "../../types";

const MINUTES_PER_DAY = HOURS_PER_DAY * MINUTES_PER_HOUR;
const MS_PER_MINUTE = 60_000;
const DRAG_SNAP_MINUTES = 30;

// A visible day column and where it is on screen.
export type DragColumn = { date: Date; rect: DOMRect };

// Own timed events and tasks can be dragged. Shared events are read-only,
// and all-day events live in the header, not the hour grid.
export const canDrag = (entry: TimelineEntry) =>
  entry.type === "task"
    ? Boolean(entry.deadline_at)
    : Boolean(
        entry.starts_at &&
        entry.ends_at &&
        !entry.shared_by &&
        !entry.is_all_day,
      );

export const snapToHalfHour = (date: Date) =>
  roundToNearestMinutes(date, { nearestTo: DRAG_SNAP_MINUTES });

// The time under a point on the grid. A point beside the grid counts as the
// nearest column, and above or below it as the start or end of the day.
export const timeFromPoint = (columns: DragColumn[], x: number, y: number) => {
  if (!columns.length) return null;
  const column =
    columns.find(({ rect }) => x < rect.right) ?? columns[columns.length - 1];
  const { top, height } = column.rect;
  const fraction = height ? Math.min(1, Math.max(0, (y - top) / height)) : 0;
  return addMilliseconds(
    startOfDay(column.date),
    fraction * MINUTES_PER_DAY * MS_PER_MINUTE,
  );
};

// Where the item was grabbed, measured from its start (a task's deadline),
// so it keeps its position under the pointer while dragged.
export const getGrabOffset = (entry: TimelineEntry, pointerTime: Date) =>
  differenceInMilliseconds(
    pointerTime,
    new Date(getTimelineEntryDate(entry) ?? pointerTime),
  );

export const getDropStart = (pointerTime: Date, grabOffset: number) =>
  snapToHalfHour(addMilliseconds(pointerTime, -grabOffset));

// A copy of the item at a new start. Events keep their duration; a task's
// deadline is its start.
export const moveEntry = (entry: TimelineEntry, start: Date): TimelineEntry => {
  if (entry.type === "task") {
    return { ...entry, deadline_at: start.toISOString() };
  }
  if (!entry.starts_at || !entry.ends_at) return entry;
  const shift = differenceInMilliseconds(start, new Date(entry.starts_at));
  return {
    ...entry,
    starts_at: start.toISOString(),
    ends_at: addMilliseconds(new Date(entry.ends_at), shift).toISOString(),
  };
};
