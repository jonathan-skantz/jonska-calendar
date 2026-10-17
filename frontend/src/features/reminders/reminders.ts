import { isSameDay, addDays, startOfDay } from "date-fns";
import type { Reminder } from "../../types";

const MINUTE = 60;
const HOUR = 60 * MINUTE;
const DAY = 24 * HOUR;

export const MAX_REMINDER_OFFSET_SECONDS = 30 * DAY;

// All-day events start at midnight, so their presets are times on earlier
// days; the backend only stores offsets before the start.
export const TIMED_REMINDER_PRESETS = [0, 5 * MINUTE, 15 * MINUTE, HOUR, DAY];
export const ALL_DAY_REMINDER_PRESETS = [
  6 * HOUR,
  15 * HOUR,
  DAY + 15 * HOUR,
  6 * DAY + 15 * HOUR,
];

const pad = (value: number) => String(value).padStart(2, "0");
const plural = (count: number, unit: string) =>
  `${count} ${unit}${count === 1 ? "" : "s"}`;

export const formatReminderOffset = (
  offsetSeconds: number,
  isAllDay = false,
) => {
  if (isAllDay) {
    if (offsetSeconds === 0) return "At midnight";
    const days = Math.ceil(offsetSeconds / DAY);
    const secondsAfterMidnight = days * DAY - offsetSeconds;
    const time = `${pad(Math.floor(secondsAfterMidnight / HOUR))}:${pad(
      Math.floor((secondsAfterMidnight % HOUR) / MINUTE),
    )}`;
    return `${days === 1 ? "Day before" : `${days} days before`} at ${time}`;
  }
  if (offsetSeconds === 0) return "At start";
  if (offsetSeconds % DAY === 0)
    return `${plural(offsetSeconds / DAY, "day")} before`;
  if (offsetSeconds % HOUR === 0) return `${offsetSeconds / HOUR} h before`;
  return `${Math.round(offsetSeconds / MINUTE)} min before`;
};

// Reminders whose time has come and that nobody has delivered yet. A skipped
// event's reminders stay quiet; an ended event's are no longer useful.
export const isReminderDue = (reminder: Reminder, now: Date) =>
  !reminder.notified_at &&
  !reminder.event_is_skipped &&
  new Date(reminder.remind_at) <= now &&
  new Date(reminder.event_ends_at) > now;

export const isReminderUpcoming = (reminder: Reminder, now: Date) =>
  !reminder.event_is_skipped && new Date(reminder.remind_at) > now;

export type ReminderGroup = {
  label: "Earlier" | "Today" | "Tomorrow" | "Later";
  reminders: Reminder[];
};

export const groupReminders = (
  reminders: Reminder[],
  now: Date,
): ReminderGroup[] => {
  const groups: ReminderGroup[] = [
    { label: "Earlier", reminders: [] },
    { label: "Today", reminders: [] },
    { label: "Tomorrow", reminders: [] },
    { label: "Later", reminders: [] },
  ];
  const tomorrow = addDays(startOfDay(now), 1);
  for (const reminder of reminders) {
    const remindAt = new Date(reminder.remind_at);
    const group =
      remindAt <= now
        ? groups[0]
        : isSameDay(remindAt, now)
          ? groups[1]
          : isSameDay(remindAt, tomorrow)
            ? groups[2]
            : groups[3];
    group.reminders.push(reminder);
  }
  return groups.filter((group) => group.reminders.length > 0);
};

export const getRemindersByEvent = (reminders: Reminder[]) => {
  const byEvent = new Map<string, Reminder[]>();
  for (const reminder of reminders) {
    byEvent.set(reminder.event_id, [
      ...(byEvent.get(reminder.event_id) ?? []),
      reminder,
    ]);
  }
  return byEvent;
};

export const describeReminders = (reminders: Reminder[]) =>
  `Reminders: ${[...reminders]
    .sort((left, right) => left.offset_seconds - right.offset_seconds)
    .map((reminder) =>
      formatReminderOffset(
        reminder.offset_seconds,
        reminder.event_is_all_day,
      ).toLowerCase(),
    )
    .join(", ")}`;
