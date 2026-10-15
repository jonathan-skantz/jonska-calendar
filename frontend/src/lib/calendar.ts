import { format, getHours, getMinutes, startOfWeek } from "date-fns";

export const WEEK_DAY_COUNT = 7;
export const WEEK_STARTS_ON = 0;
export const HOURS_PER_DAY = 24;
export const MINUTES_PER_HOUR = 60;
export const CALENDAR_GRID_SCALE = 2;
export const TASK_DISPLAY_DURATION_MINUTES = 45;
export const MINIMUM_ENTRY_DURATION_MINUTES = 90;
export const PREVIEW_EVENT_DURATION_MINUTES = 120;
export const CLOCK_UPDATE_INTERVAL_MS = 60_000;
export const EVENT_SEPARATOR_PIXELS = 1;
export const EVENT_OVERLAP_INSET_PIXELS = 6;
export const LAST_EVENT_START_HOUR = 22;

export const formatTime = (value?: string) =>
  value ? format(new Date(value), "p") : "";
export const formatDate = (value: Date) => format(value, "EEE, MMM d");
export const dateKey = (value: Date) => format(value, "yyyy-MM-dd");
export const inputDateTime = (value?: string) =>
  value ? format(new Date(value), "yyyy-MM-dd'T'HH:mm") : "";
export const startOfCalendarWeek = (value: Date) =>
  startOfWeek(value, { weekStartsOn: WEEK_STARTS_ON });

export const localDateTime = (value: FormDataEntryValue | null) => {
  if (typeof value !== "string" || !value) return undefined;
  return new Date(value).toISOString();
};

export const localDateKey = (value?: string) =>
  value ? dateKey(new Date(value)) : "";

export const minutesIntoDay = (value?: string) => {
  if (!value) return 0;
  const date = new Date(value);
  return getHours(date) * MINUTES_PER_HOUR + getMinutes(date);
};

export const sameInstant = (left?: string, right?: string) =>
  Boolean(left && right && new Date(left).getTime() === new Date(right).getTime());
