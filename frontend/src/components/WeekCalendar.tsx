import type { MouseEvent, ReactNode } from "react";
import { dateKey, formatDate, formatHour } from "../lib/calendar";
import type { TimeFormatPreference } from "../lib/calendar";

type WeekCalendarProps = {
  dates: Date[];
  onDayClick: (event: MouseEvent<HTMLDivElement>, date: Date) => void;
  renderDay: (date: Date) => ReactNode;
  renderAllDay: () => ReactNode;
  transitCardPills: ReactNode;
  onTransitCardClick?: () => void;
  hasAllDayEvents: boolean;
  timeFormat: TimeFormatPreference;
};

export const WeekCalendar = ({
  dates,
  onDayClick,
  renderDay,
  renderAllDay,
  transitCardPills,
  onTransitCardClick,
  hasAllDayEvents,
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
    <div className="all-day-row transit-card-row">
      <span className="all-day-label">Card</span>
      <button
        type="button"
        className="transit-card-grid"
        aria-label="Edit transit card days"
        disabled={!onTransitCardClick}
        onClick={onTransitCardClick}
      >
        {transitCardPills}
      </button>
    </div>
    {hasAllDayEvents && (
      <div className="all-day-row">
        <span className="all-day-label">All day</span>
        <div className="all-day-grid">{renderAllDay()}</div>
      </div>
    )}
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
