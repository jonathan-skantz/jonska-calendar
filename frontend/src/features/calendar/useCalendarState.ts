import { useState } from "react";
import { addDays } from "date-fns";
import {
  dateKey,
  getWeekStartsOn,
  startOfCalendarWeek,
} from "../../lib/calendar";
import type { WeekStartsOn } from "../../lib/calendar";

export const useCalendarState = (
  today: Date,
  weekStartsOn: WeekStartsOn = getWeekStartsOn(),
) => {
  const [weekView, setWeekView] = useState(() => ({
    weekStartsOn,
    weekStart: startOfCalendarWeek(today, weekStartsOn),
  }));
  const weekStart =
    weekView.weekStartsOn === weekStartsOn
      ? weekView.weekStart
      : startOfCalendarWeek(addDays(weekView.weekStart, 3), weekStartsOn);
  const [selectedDate, setSelectedDate] = useState(() => dateKey(today));

  const moveWeek = (days: number) => {
    const nextWeek = addDays(weekStart, days);
    setWeekView({ weekStartsOn, weekStart: nextWeek });
    setSelectedDate(dateKey(nextWeek));
  };

  return { weekStart, selectedDate, setSelectedDate, moveWeek };
};
