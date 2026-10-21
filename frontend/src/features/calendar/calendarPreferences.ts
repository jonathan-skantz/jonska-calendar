import { getWeekStartsOn } from "../../lib/calendar";
import type { TimeFormatPreference, WeekStartsOn } from "../../lib/calendar";

export type WeekStartPreference = "browser" | WeekStartsOn;

export type SunDisplayPreferences = {
  showSunrise: boolean;
  showSunset: boolean;
  showNight: boolean;
};

// How many hours of the day fill the week grid's height; the rest scrolls.
export const VISIBLE_HOUR_OPTIONS = [24, 16, 12] as const;
export type VisibleHours = (typeof VISIBLE_HOUR_OPTIONS)[number];

export type CalendarPreferences = SunDisplayPreferences & {
  timeFormat: TimeFormatPreference;
  weekStartsOn: WeekStartPreference;
  visibleHours: VisibleHours;
};

export const DEFAULT_CALENDAR_PREFERENCES: CalendarPreferences = {
  timeFormat: "browser",
  weekStartsOn: "browser",
  showSunrise: true,
  showSunset: true,
  showNight: true,
  visibleHours: 24,
};

const booleanOr = (value: unknown, fallback: boolean) =>
  typeof value === "boolean" ? value : fallback;

const isTimeFormatPreference = (
  value: unknown,
): value is TimeFormatPreference =>
  value === "browser" || value === "12-hour" || value === "24-hour";

const isWeekStartPreference = (value: unknown): value is WeekStartPreference =>
  value === "browser" ||
  (typeof value === "number" &&
    Number.isInteger(value) &&
    value >= 0 &&
    value <= 6);

const isVisibleHours = (value: unknown): value is VisibleHours =>
  VISIBLE_HOUR_OPTIONS.includes(value as VisibleHours);

export const parseCalendarPreferences = (
  value: string | null,
): CalendarPreferences => {
  if (!value) return DEFAULT_CALENDAR_PREFERENCES;
  try {
    const parsed: unknown = JSON.parse(value);
    if (!parsed || typeof parsed !== "object")
      return DEFAULT_CALENDAR_PREFERENCES;
    const preferences = parsed as Partial<CalendarPreferences>;
    return {
      timeFormat: isTimeFormatPreference(preferences.timeFormat)
        ? preferences.timeFormat
        : "browser",
      weekStartsOn: isWeekStartPreference(preferences.weekStartsOn)
        ? preferences.weekStartsOn
        : "browser",
      showSunrise: booleanOr(preferences.showSunrise, true),
      showSunset: booleanOr(preferences.showSunset, true),
      showNight: booleanOr(preferences.showNight, true),
      visibleHours: isVisibleHours(preferences.visibleHours)
        ? preferences.visibleHours
        : 24,
    };
  } catch {
    return DEFAULT_CALENDAR_PREFERENCES;
  }
};

export const getEffectiveWeekStart = (
  preference: WeekStartPreference,
): WeekStartsOn => (preference === "browser" ? getWeekStartsOn() : preference);

const WEEKDAY_LABEL_REFERENCE_SUNDAY = new Date(2020, 0, 5);

export const weekdayName = (day: WeekStartsOn, locale?: string) => {
  const date = new Date(WEEKDAY_LABEL_REFERENCE_SUNDAY);
  date.setDate(date.getDate() + day);
  return new Intl.DateTimeFormat(locale, { weekday: "long" }).format(date);
};
