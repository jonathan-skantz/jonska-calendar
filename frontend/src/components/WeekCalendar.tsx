import type { MouseEvent, ReactNode } from "react";
import { dateKey, formatDate, formatHour } from "../lib/calendar";
import type { TimeFormatPreference } from "../lib/calendar";

type WeekCalendarProps = {
  dates: Date[];
  onDayClick: (event: MouseEvent<HTMLDivElement>, date: Date) => void;
  renderDay: (date: Date) => ReactNode;
  timeFormat: TimeFormatPreference;
};

export const WeekCalendar = ({
  dates,
  onDayClick,
  renderDay,
  timeFormat,
}: WeekCalendarProps) => (
  <div className="week-calendar" aria-label="Week calendar">
    <div className="week-day-headings">
      <span />
      {dates.map((date) => (
        <div
          className={`day-heading ${date.getDay() === 0 || date.getDay() === 6 ? "weekend" : ""}`}
          key={dateKey(date)}
        >
          <strong>{date.getDate()}</strong>
          <span>{formatDate(date).split(",")[0]}</span>
        </div>
      ))}
    </div>
    <div className="week-grid">
      <div className="hour-labels">
        {Array.from({ length: 24 }, (_, hour) => (
          <span key={hour}>{formatHour(hour, { preference: timeFormat })}</span>
        ))}
      </div>
      {dates.map((date) => (
        <div
          className={`day-column ${date.getDay() === 0 || date.getDay() === 6 ? "weekend" : ""}`}
          key={dateKey(date)}
          onClick={(event) => onDayClick(event, date)}
        >
          {renderDay(date)}
        </div>
      ))}
    </div>
  </div>
);
