import { addDays, format, parseISO } from "date-fns";
import type { Recurrence, TimelineEntry, Weekday } from "../types";

// Monday first, as the backend validates and orders them.
export const WEEKDAYS: Weekday[] = ["MO", "TU", "WE", "TH", "FR", "SA", "SU"];

export const MAX_REPEAT_INTERVAL = 99;
export const MAX_REPEAT_COUNT = 999;

const ORDINALS = ["1st", "2nd", "3rd", "4th"];

const UNITS: Record<Recurrence["frequency"], string> = {
  daily: "day",
  weekly: "week",
  monthly: "month",
  yearly: "year",
};

export const weekdayOf = (date: Date): Weekday =>
  WEEKDAYS[(date.getDay() + 6) % 7];

// Which of its weekday in the month a date is: 1 for the first Tuesday, 5
// for a fifth.
export const weekOfMonth = (date: Date) =>
  Math.floor((date.getDate() - 1) / 7) + 1;

export const isLastWeekdayOfMonth = (date: Date) =>
  addDays(date, 7).getMonth() !== date.getMonth();

const weekdayName = (date: Date) => format(date, "EEEE");

// The ways a monthly repeat can pick its day for a series starting on date.
export const monthlyOptions = (date: Date) => {
  const options: {
    value: NonNullable<Recurrence["monthly_by"]>;
    label: string;
  }[] = [{ value: "day_of_month", label: `On day ${date.getDate()}` }];
  if (weekOfMonth(date) <= 4) {
    options.push({
      value: "weekday",
      label: `On the ${ORDINALS[weekOfMonth(date) - 1]} ${weekdayName(date)}`,
    });
  }
  if (isLastWeekdayOfMonth(date)) {
    options.push({
      value: "last_weekday",
      label: `On the last ${weekdayName(date)}`,
    });
  }
  return options;
};

const WEEKDAY_LABELS: Record<Weekday, string> = {
  MO: "Mon",
  TU: "Tue",
  WE: "Wed",
  TH: "Thu",
  FR: "Fri",
  SA: "Sat",
  SU: "Sun",
};

export const weekdayLabel = (weekday: Weekday) => WEEKDAY_LABELS[weekday];

// A short description, such as "Every 2 weeks on Tue, Thu, 10 times".
export const describeRecurrence = (rule: Recurrence, start: Date) => {
  const unit = UNITS[rule.frequency];
  let text =
    rule.interval === 1
      ? `${rule.frequency[0].toUpperCase()}${rule.frequency.slice(1)}`
      : `Every ${rule.interval} ${unit}s`;
  if (rule.frequency === "weekly" && rule.weekdays?.length) {
    text += ` on ${rule.weekdays.map(weekdayLabel).join(", ")}`;
  }
  if (rule.frequency === "monthly") {
    const option = monthlyOptions(start).find(
      ({ value }) => value === (rule.monthly_by ?? "day_of_month"),
    );
    if (option) text += ` o${option.label.slice(1)}`;
  }
  if (rule.until)
    text += `, until ${format(parseISO(rule.until), "MMM d, yyyy")}`;
  if (rule.count)
    text += `, ${rule.count} ${rule.count === 1 ? "time" : "times"}`;
  return text;
};

// How an occurrence's event repeats, or undefined for a plain event.
export const describeEntryRecurrence = (entry: TimelineEntry) =>
  entry.recurrence && entry.starts_at
    ? describeRecurrence(entry.recurrence, new Date(entry.starts_at))
    : undefined;

// The repeat the event form describes, or null when it does not repeat.
export const readRecurrence = (form: FormData): Recurrence | null => {
  const frequency = form.get("repeat_frequency");
  if (
    frequency !== "daily" &&
    frequency !== "weekly" &&
    frequency !== "monthly" &&
    frequency !== "yearly"
  ) {
    return null;
  }
  const rule: Recurrence = {
    frequency,
    interval: Number(form.get("repeat_interval") || 1),
  };
  if (frequency === "weekly") {
    rule.weekdays = WEEKDAYS.filter((day) =>
      form.getAll("repeat_weekday").includes(day),
    );
  }
  if (frequency === "monthly") {
    rule.monthly_by = (form.get("repeat_monthly_by") ??
      "day_of_month") as Recurrence["monthly_by"];
  }
  const ends = form.get("repeat_ends");
  if (ends === "on") rule.until = String(form.get("repeat_until"));
  if (ends === "after") rule.count = Number(form.get("repeat_count"));
  return rule;
};
