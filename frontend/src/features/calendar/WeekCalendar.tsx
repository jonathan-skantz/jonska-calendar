import { useLayoutEffect, useRef } from "react";
import type { CSSProperties, MouseEvent, PointerEvent, ReactNode } from "react";
import {
  HOURS_PER_DAY,
  dateKey,
  formatDate,
  formatHour,
} from "../../lib/calendar";
import {
  SWIPE_MAX_VERTICAL_RATIO,
  SWIPE_MIN_DISTANCE_PX,
} from "../../lib/layout";
import type { WeatherDay } from "../../types";
import { WeatherIcons } from "../weather/WeatherIcons";
import { cx } from "../../lib/classNames";
import { usePreferences } from "./preferencesContext";

type WeekCalendarProps = {
  dates: Date[];
  today: Date;
  onDayClick: (event: MouseEvent<HTMLDivElement>, date: Date) => void;
  renderDay: (date: Date) => ReactNode;
  renderAllDay: () => ReactNode;
  transitCardPills: ReactNode;
  onTransitCardClick?: () => void;
  hasAllDayEvents: boolean;
  weatherByDay: Map<string, WeatherDay>;
  // Hour scrolled into view whenever a new date range is shown.
  scrollToHour: number;
  // Horizontal swipe on touch screens; -1 = previous range, 1 = next.
  onSwipe?: (direction: 1 | -1) => void;
};

const isWeekend = (date: Date) => date.getDay() === 0 || date.getDay() === 6;

export const WeekCalendar = ({
  dates,
  today,
  onDayClick,
  renderDay,
  renderAllDay,
  transitCardPills,
  onTransitCardClick,
  hasAllDayEvents,
  weatherByDay,
  scrollToHour,
  onSwipe,
}: WeekCalendarProps) => {
  const { timeFormat, visibleHours } = usePreferences().preferences;
  const scrollerRef = useRef<HTMLDivElement>(null);
  const headerRef = useRef<HTMLDivElement>(null);
  const gridRef = useRef<HTMLDivElement>(null);
  const swipeStart = useRef<{ x: number; y: number } | null>(null);
  const rangeKey = `${dateKey(dates[0])}/${dates.length}`;

  // The hour height comes from the space left under the sticky header. It is
  // measured rather than estimated because the header grows with all-day rows
  // and wrapping, and it runs before the scroll effect below so that sees the
  // final grid height.
  useLayoutEffect(() => {
    const scroller = scrollerRef.current;
    const header = headerRef.current;
    if (!scroller || !header) return;
    const update = () =>
      scroller.style.setProperty(
        "--grid-available-height",
        `${Math.floor(scroller.clientHeight - header.offsetHeight)}px`,
      );
    update();
    if (typeof ResizeObserver === "undefined") return;
    const observer = new ResizeObserver(update);
    observer.observe(scroller);
    observer.observe(header);
    return () => observer.disconnect();
  }, []);

  // Scroll on range change only, so the grid doesn't jump as the clock ticks.
  // The sticky header is in flow above the grid, so scrollTop maps directly to
  // the grid offset of the first hour visible under it.
  const scrolledRangeKey = useRef<string | null>(null);
  useLayoutEffect(() => {
    const scroller = scrollerRef.current;
    const grid = gridRef.current;
    if (!scroller || !grid || scrolledRangeKey.current === rangeKey) return;
    scrolledRangeKey.current = rangeKey;
    scroller.scrollTop = (scrollToHour / HOURS_PER_DAY) * grid.offsetHeight;
  }, [rangeKey, scrollToHour]);

  // Zooming changes the grid height; keep the same hour at the top. The
  // fraction is recorded on scroll because the grid has already resized by
  // the time the effect runs.
  const topFraction = useRef(0);
  const zoomedHours = useRef(visibleHours);
  const recordScroll = () => {
    const scroller = scrollerRef.current;
    const grid = gridRef.current;
    if (scroller && grid?.offsetHeight)
      topFraction.current = scroller.scrollTop / grid.offsetHeight;
  };
  useLayoutEffect(() => {
    const scroller = scrollerRef.current;
    const grid = gridRef.current;
    if (!scroller || !grid || zoomedHours.current === visibleHours) return;
    zoomedHours.current = visibleHours;
    scroller.scrollTop = topFraction.current * grid.offsetHeight;
  }, [visibleHours]);

  const handlePointerDown = (event: PointerEvent<HTMLDivElement>) => {
    swipeStart.current =
      onSwipe && event.pointerType === "touch"
        ? { x: event.clientX, y: event.clientY }
        : null;
  };

  const handlePointerUp = (event: PointerEvent<HTMLDivElement>) => {
    const start = swipeStart.current;
    swipeStart.current = null;
    if (!start || !onSwipe) return;
    const dx = event.clientX - start.x;
    const dy = event.clientY - start.y;
    if (
      Math.abs(dx) >= SWIPE_MIN_DISTANCE_PX &&
      Math.abs(dy) <= Math.abs(dx) * SWIPE_MAX_VERTICAL_RATIO
    ) {
      onSwipe(dx < 0 ? 1 : -1);
    }
  };

  return (
    <div
      ref={scrollerRef}
      className={cx("week-calendar", onSwipe && "swipeable")}
      aria-label="Week calendar"
      style={
        {
          "--day-count": dates.length,
          "--visible-hours": visibleHours,
        } as CSSProperties
      }
      onScroll={recordScroll}
      onPointerDown={handlePointerDown}
      onPointerUp={handlePointerUp}
      onPointerCancel={() => (swipeStart.current = null)}
    >
      <div className="week-calendar-header" ref={headerRef}>
        <div className="week-day-headings">
          <span />
          {dates.map((date) => (
            <div
              className={cx(
                "day-heading",
                isWeekend(date) && "weekend",
                dateKey(date) === dateKey(today) && "today",
              )}
              aria-current={
                dateKey(date) === dateKey(today) ? "date" : undefined
              }
              key={dateKey(date)}
            >
              <strong>{date.getDate()}</strong>
              <span>{formatDate(date).split(",")[0]}</span>
              <WeatherIcons
                conditions={weatherByDay.get(dateKey(date))?.conditions ?? []}
              />
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
      </div>
      <div className="week-grid" ref={gridRef}>
        <div className="hour-labels">
          {Array.from({ length: HOURS_PER_DAY }, (_, hour) => (
            <span key={hour}>
              {formatHour(hour, { preference: timeFormat })}
            </span>
          ))}
        </div>
        {dates.map((date) => (
          <div
            className={cx("day-column", isWeekend(date) && "weekend")}
            key={dateKey(date)}
            data-date={dateKey(date)}
            onClick={(event) => onDayClick(event, date)}
          >
            {renderDay(date)}
          </div>
        ))}
      </div>
    </div>
  );
};
