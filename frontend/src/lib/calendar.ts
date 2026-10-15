import { format, getHours, getMinutes, startOfWeek } from "date-fns";

export const WEEK_DAY_COUNT = 7;
export const HOURS_PER_DAY = 24;
export const MINUTES_PER_HOUR = 60;
export const CALENDAR_GRID_SCALE = 2;
export const TASK_DISPLAY_DURATION_MINUTES = 45;
export const MINIMUM_ENTRY_DURATION_MINUTES = 90;
export const DEFAULT_EVENT_DURATION_MINUTES = 120;
export const CLOCK_UPDATE_INTERVAL_MS = 60_000;
export const EVENT_SEPARATOR_PIXELS = 1;
export const EVENT_OVERLAP_INSET_PIXELS = 6;
export const LAST_EVENT_START_HOUR = 22;

type LocaleWeekInfo = { firstDay: number };
type WeekStartsOn = 0 | 1 | 2 | 3 | 4 | 5 | 6;
type LocaleWithWeekInfo = Intl.Locale & {
  getWeekInfo?: () => LocaleWeekInfo;
  weekInfo?: LocaleWeekInfo;
};

const getLocaleFirstDay = (localeName: string) => {
  if (typeof Intl.Locale === "undefined") return undefined;
  const locale = new Intl.Locale(localeName) as LocaleWithWeekInfo;
  return locale.getWeekInfo?.().firstDay ?? locale.weekInfo?.firstDay;
};

export const getWeekStartsOn = (
  localeName = Intl.DateTimeFormat().resolvedOptions().locale,
): WeekStartsOn => {
  const firstDay = getLocaleFirstDay(localeName);
  return firstDay === undefined ? 1 : ((firstDay % 7) as WeekStartsOn);
};

const formatTimeValue = (value: Date, locale?: string) => {
  if (typeof Intl.DateTimeFormat === "function") {
    return new Intl.DateTimeFormat(locale, {
      hour: "numeric",
      minute: "2-digit",
    }).format(value);
  }
  return `${String(value.getHours()).padStart(2, "0")}:${String(value.getMinutes()).padStart(2, "0")}`;
};

export const formatTime = (value?: string | Date, locale?: string) => {
  if (!value) return "";
  return formatTimeValue(
    value instanceof Date ? value : new Date(value),
    locale,
  );
};

export const formatHour = (hour: number, locale?: string) =>
  formatTimeValue(new Date(2020, 0, 1, hour), locale);
export const formatDate = (value: Date) => format(value, "EEE, MMM d");
export const dateKey = (value: Date) => format(value, "yyyy-MM-dd");
export const inputDateTime = (value?: string) =>
  value ? format(new Date(value), "yyyy-MM-dd'T'HH:mm") : "";
export const startOfCalendarWeek = (value: Date) =>
  startOfWeek(value, { weekStartsOn: getWeekStartsOn() });

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
  Boolean(
    left && right && new Date(left).getTime() === new Date(right).getTime(),
  );
