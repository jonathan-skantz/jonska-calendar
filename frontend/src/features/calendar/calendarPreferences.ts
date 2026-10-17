import { getWeekStartsOn } from "../../lib/calendar";
import type { TimeFormatPreference, WeekStartsOn } from "../../lib/calendar";

export type WeekStartPreference = "browser" | WeekStartsOn;

export type SunDisplayPreferences = {
  showSunrise: boolean;
  showSunset: boolean;
  showNight: boolean;
};

export type CalendarPreferences = SunDisplayPreferences & {
  timeFormat: TimeFormatPreference;
  weekStartsOn: WeekStartPreference;
};

export const DEFAULT_CALENDAR_PREFERENCES: CalendarPreferences = {
  timeFormat: "browser",
  weekStartsOn: "browser",
  showSunrise: true,
  showSunset: true,
  showNight: true,
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
