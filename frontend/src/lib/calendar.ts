import { format, getHours, getMinutes, startOfWeek } from "date-fns";

export const WEEK_DAY_COUNT = 7;
// Days shown at once on phone-width screens.
export const PHONE_DAY_COUNT = 3;
// Hour the week grid scrolls to when it first renders a range.
export const DEFAULT_SCROLL_HOUR = 7;
export const HOURS_PER_DAY = 24;
export const MINUTES_PER_HOUR = 60;
export const SECONDS_PER_MINUTE = 60;
export const MAX_GATHERING_LABEL_LENGTH = 40;
export const MAX_COMMUTE_MINUTES = 24 * 60;
export const TASK_DISPLAY_DURATION_MINUTES = 45;
export const MINIMUM_ENTRY_DURATION_MINUTES = 90;
export const DEFAULT_EVENT_DURATION_MINUTES = 120;
export const CLOCK_UPDATE_INTERVAL_MS = 60_000;
export const EVENT_SEPARATOR_PIXELS = 1;
export const EVENT_OVERLAP_INSET_PIXELS = 6;
export const LAST_EVENT_START_HOUR = 22;

// Vertical grid positions scale with the --hour-height CSS variable so the
// week grid can grow with the viewport (see styles/tokens.css).
export const gridOffset = (minutes: number) =>
  `calc(${minutes / MINUTES_PER_HOUR} * var(--hour-height))`;

type LocaleWeekInfo = { firstDay: number };
export type WeekStartsOn = 0 | 1 | 2 | 3 | 4 | 5 | 6;
export type TimeFormatPreference = "browser" | "12-hour" | "24-hour";
export type TimeFormatOptions = {
  locale?: string;
  preference?: TimeFormatPreference;
};

export const getEffectiveTimeFormat = (
  preference: TimeFormatPreference,
): Exclude<TimeFormatPreference, "browser"> => {
  if (preference !== "browser") return preference;
  const hourCycle = new Intl.DateTimeFormat(undefined, {
    hour: "numeric",
  }).resolvedOptions().hourCycle;
  return hourCycle === "h11" || hourCycle === "h12" ? "12-hour" : "24-hour";
};
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

const formatTimeValue = (
  value: Date,
  { locale, preference = "browser" }: TimeFormatOptions = {},
) => {
  if (typeof Intl.DateTimeFormat === "function") {
    const hourCycle =
      preference === "12-hour"
        ? "h12"
        : preference === "24-hour"
          ? "h23"
          : undefined;
    const formatter = new Intl.DateTimeFormat(locale, {
      hour: "numeric",
      minute: "2-digit",
      ...(hourCycle ? { hourCycle } : {}),
    });
    return formatter.format(value);
  }
  if (preference === "12-hour") {
    const hour = value.getHours();
    return `${hour % 12 || 12}:${String(value.getMinutes()).padStart(2, "0")} ${hour < 12 ? "AM" : "PM"}`;
  }
  return `${String(value.getHours()).padStart(2, "0")}:${String(value.getMinutes()).padStart(2, "0")}`;
};

export const formatTime = (
  value?: string | Date,
  options?: TimeFormatOptions,
) => {
  if (!value) return "";
  return formatTimeValue(
    value instanceof Date ? value : new Date(value),
    options,
  );
};

// Approximate event edges are shown as "~10:00"; the stored time stays exact.
export const formatEventTime = (
  value: string | Date | undefined,
  isApproximate: boolean | undefined,
  options?: TimeFormatOptions,
) => {
  const time = formatTime(value, options);
  return time && isApproximate ? `~${time}` : time;
};

export const formatHour = (hour: number, options?: TimeFormatOptions) =>
  formatTimeValue(new Date(2020, 0, 1, hour), options);
export const formatDate = (value: Date) => format(value, "EEE, MMM d");
export const formatShortDate = (value: Date) => format(value, "MMM d");
export const dateKey = (value: Date) => format(value, "yyyy-MM-dd");
export const inputDateTime = (value?: string) =>
  value ? format(new Date(value), "yyyy-MM-dd'T'HH:mm") : "";
export const startOfCalendarWeek = (
  value: Date,
  weekStartsOn = getWeekStartsOn(),
) => startOfWeek(value, { weekStartsOn });

export const localDateTime = (value: FormDataEntryValue | null) => {
  if (typeof value !== "string" || !value) return undefined;
  return new Date(value).toISOString();
};

export const localDateKey = (value?: string) =>
  value ? dateKey(new Date(value)) : "";

export const minutesIntoDay = (value?: string | Date) => {
  if (!value) return 0;
  const date = value instanceof Date ? value : new Date(value);
  return getHours(date) * MINUTES_PER_HOUR + getMinutes(date);
};

export const sameInstant = (left?: string, right?: string) =>
  Boolean(
    left && right && new Date(left).getTime() === new Date(right).getTime(),
  );
