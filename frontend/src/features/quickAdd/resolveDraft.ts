import {
  addDays,
  addMinutes,
  addWeeks,
  format,
  isBefore,
  startOfDay,
  startOfWeek,
} from "date-fns";
import {
  DEFAULT_EVENT_DURATION_MINUTES,
  SECONDS_PER_MINUTE,
  type WeekStartsOn,
} from "../../lib/calendar";
import type { DraftDate, EventDraft, TimelineEntry } from "../../types";

export type ResolvedDraft = {
  // Shaped like a saved event so the composer can use it as initial values.
  values: Partial<TimelineEntry>;
  tagIds: string[];
  reminderOffsets: number[];
  warnings: string[];
};

const DEFAULT_START_TIME = "09:00";

const WEEKDAYS = [
  "sunday",
  "monday",
  "tuesday",
  "wednesday",
  "thursday",
  "friday",
  "saturday",
] as const;

// Resolves a stated date to a local day, or null when it does not exist
// (e.g. February 30).
export const resolveDate = (
  spec: DraftDate,
  today: Date,
  weekStartsOn: WeekStartsOn,
): Date | null => {
  const day = startOfDay(today);
  switch (spec.kind) {
    case "relative_days":
      return addDays(day, spec.days);
    case "weekday": {
      const target = WEEKDAYS.indexOf(spec.weekday);
      // "Friday" is the next one, today included. "Next Friday" is the one in
      // the following calendar week, which depends on the week start.
      if (!spec.week_offset)
        return addDays(day, (target - day.getDay() + 7) % 7);
      const nextWeek = startOfWeek(addWeeks(day, 1), { weekStartsOn });
      return addDays(nextWeek, (target - weekStartsOn + 7) % 7);
    }
    case "calendar": {
      const build = (year: number) => {
        const date = new Date(year, spec.month - 1, spec.day);
        return date.getMonth() === spec.month - 1 ? date : null;
      };
      if (spec.year !== undefined) return build(spec.year);
      // Without a year, the next time that date comes around.
      const thisYear = build(day.getFullYear());
      if (thisYear && !isBefore(thisYear, day)) return thisYear;
      return build(day.getFullYear() + 1);
    }
  }
};

const atClock = (date: Date, clock: string) => {
  const [hours, minutes] = clock.split(":").map(Number);
  const result = new Date(date);
  result.setHours(hours, minutes, 0, 0);
  return result;
};

const toSeconds = (minutes?: number) =>
  minutes === undefined ? undefined : minutes * SECONDS_PER_MINUTE;

export const resolveDraft = (
  draft: EventDraft,
  today: Date,
  weekStartsOn: WeekStartsOn,
  selectedDate: string,
): ResolvedDraft => {
  const warnings: string[] = [];
  const todayStart = startOfDay(today);

  let day = draft.date ? resolveDate(draft.date, today, weekStartsOn) : null;
  if (draft.date && !day) warnings.push("That date doesn't exist");
  if (!draft.date) warnings.push("No date given, so it uses the selected day");
  day ??= new Date(`${selectedDate}T00:00`);
  if (isBefore(day, todayStart)) warnings.push("That date has passed");

  let lastDay = draft.end_date
    ? resolveDate(draft.end_date, today, weekStartsOn)
    : null;
  if (draft.end_date && (!lastDay || isBefore(lastDay, day))) {
    warnings.push("Couldn't use the end date");
    lastDay = null;
  }

  const isAllDay = Boolean(draft.all_day);
  let start: Date;
  let end: Date;
  if (isAllDay) {
    start = day;
    // All-day events end at midnight after their last day.
    end = addDays(lastDay ?? day, 1);
  } else {
    if (!draft.start_time)
      warnings.push(`No time given, so it starts at ${DEFAULT_START_TIME}`);
    start = atClock(day, draft.start_time ?? DEFAULT_START_TIME);
    const fallbackEnd = addMinutes(
      start,
      draft.duration_minutes ?? DEFAULT_EVENT_DURATION_MINUTES,
    );
    if (draft.end_time) {
      end = atClock(lastDay ?? day, draft.end_time);
      if (!lastDay && end <= start) {
        end = addDays(end, 1);
        warnings.push(`Ends after midnight, on ${format(end, "EEEE d MMMM")}`);
      }
    } else if (lastDay) {
      end = atClock(lastDay, format(fallbackEnd, "HH:mm"));
    } else {
      end = fallbackEnd;
    }
    if (end <= start) end = fallbackEnd;
  }

  const values: Partial<TimelineEntry> = {
    title: draft.title,
    starts_at: start.toISOString(),
    ends_at: end.toISOString(),
    is_all_day: isAllDay,
    start_is_approximate: !isAllDay && draft.start_is_approximate,
    end_is_approximate: !isAllDay && draft.end_is_approximate,
    location: draft.location,
    kind: draft.kind,
    commute_before_seconds: toSeconds(draft.commute_before_minutes),
    commute_after_seconds: toSeconds(draft.commute_after_minutes),
    gathering_seconds: toSeconds(draft.gathering_minutes),
    gathering_is_optional: draft.gathering_is_optional,
    gathering_label: draft.gathering_label,
  };

  return {
    values,
    tagIds: draft.tag_ids ?? [],
    reminderOffsets: [
      ...new Set(
        (draft.reminder_minutes ?? []).map(
          (minutes) => minutes * SECONDS_PER_MINUTE,
        ),
      ),
    ],
    warnings,
  };
};
