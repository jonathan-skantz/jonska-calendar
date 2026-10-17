import { useState } from "react";
import { addDays, differenceInCalendarDays, startOfDay } from "date-fns";
import {
  WEEK_DAY_COUNT,
  dateKey,
  getWeekStartsOn,
  startOfCalendarWeek,
} from "../../lib/calendar";
import type { WeekStartsOn } from "../../lib/calendar";

type CalendarRange = {
  weekStartsOn: WeekStartsOn;
  dayCount: number;
  rangeStart: Date;
};

// A full week snaps to the calendar week containing the anchor; shorter
// ranges (the phone view) start at the anchor day itself.
export const getRangeStart = (
  anchor: Date,
  weekStartsOn: WeekStartsOn,
  dayCount: number,
) =>
  dayCount === WEEK_DAY_COUNT
    ? startOfCalendarWeek(anchor, weekStartsOn)
    : startOfDay(anchor);

// The day to keep in view when the range shape changes: today if it is
// visible, otherwise the middle of the current range.
export const getRangeAnchor = (
  range: Pick<CalendarRange, "rangeStart" | "dayCount">,
  today: Date,
) => {
  const offset = differenceInCalendarDays(today, range.rangeStart);
  return offset >= 0 && offset < range.dayCount
    ? today
    : addDays(range.rangeStart, Math.floor(range.dayCount / 2));
};

export const useCalendarState = (
  today: Date,
  weekStartsOn: WeekStartsOn = getWeekStartsOn(),
  dayCount: number = WEEK_DAY_COUNT,
) => {
  const [range, setRange] = useState<CalendarRange>(() => ({
    weekStartsOn,
    dayCount,
    rangeStart: getRangeStart(today, weekStartsOn, dayCount),
  }));
  const rangeStart =
    range.weekStartsOn === weekStartsOn && range.dayCount === dayCount
      ? range.rangeStart
      : getRangeStart(getRangeAnchor(range, today), weekStartsOn, dayCount);
  const [selectedDate, setSelectedDate] = useState(() => dateKey(today));

  const moveRange = (direction: 1 | -1) => {
    const next = addDays(rangeStart, direction * dayCount);
    setRange({ weekStartsOn, dayCount, rangeStart: next });
    setSelectedDate(dateKey(next));
  };

  return { rangeStart, selectedDate, setSelectedDate, moveRange };
};
