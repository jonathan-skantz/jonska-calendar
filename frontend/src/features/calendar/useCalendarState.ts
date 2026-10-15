import { useState } from "react";
import { addDays } from "date-fns";
import { dateKey, startOfCalendarWeek } from "../../lib/calendar";

export const useCalendarState = (today: Date) => {
  const [weekStart, setWeekStart] = useState(() => startOfCalendarWeek(today));
  const [selectedDate, setSelectedDate] = useState(() => dateKey(today));

  const moveWeek = (days: number) => {
    const nextWeek = addDays(weekStart, days);
    setWeekStart(nextWeek);
    setSelectedDate(dateKey(nextWeek));
  };

  return { weekStart, selectedDate, setSelectedDate, moveWeek };
};
