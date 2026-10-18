import { useEffect, useRef, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { formatTime } from "../../lib/calendar";
import type { TimeFormatPreference } from "../../lib/calendar";
import type { Reminder } from "../../types";
import { usePreferences } from "../calendar/preferencesContext";
import { isReminderDue } from "./reminders";
import { remindersQueryOptions, useReminderMutations } from "./useReminders";

// setTimeout overflows above ~24.8 days; later reminders are scheduled after
// a later refetch or reload instead.
const MAX_TIMEOUT_MS = 2 ** 31 - 1;
// A reminder that was already due when the app loaded was missed while no tab
// was open, rather than delivered a moment late.
const MISSED_GRACE_MS = 60 * 1000;

export type ReminderToast = {
  reminder: Reminder;
  missed: boolean;
};

const reminderBody = (reminder: Reminder, timeFormat: TimeFormatPreference) =>
  [
    reminder.event_is_all_day
      ? "All day"
      : formatTime(reminder.event_starts_at, { preference: timeFormat }),
    reminder.event_location,
  ]
    .filter(Boolean)
    .join(" · ");

// Delivers reminders while the app is open: a system notification when the
// user allowed them, otherwise an in-app toast. Missed reminders always become
// toasts. Each delivery is recorded on the server so other tabs and devices
// skip it; the tag also dedupes system notifications within one browser.
export const useReminderNotifications = (
  reminders: Reminder[],
  markNotified: (reminderId: string) => Promise<unknown>,
  timeFormat: TimeFormatPreference,
) => {
  const [toasts, setToasts] = useState<ReminderToast[]>([]);
  const delivered = useRef(new Set<string>());
  const [mountedAt] = useState(() => Date.now());
  // The scheduling effect only reruns when the reminders change; these are
  // read when a reminder fires.
  const latest = useRef({ markNotified, timeFormat });
  useEffect(() => {
    latest.current = { markNotified, timeFormat };
  });

  useEffect(() => {
    const deliver = (reminder: Reminder) => {
      // Keyed by time too, so a moved event's re-armed reminder fires again.
      const key = `${reminder.id}@${reminder.remind_at}`;
      if (delivered.current.has(key)) return;
      delivered.current.add(key);
      const missed =
        new Date(reminder.remind_at).getTime() < mountedAt - MISSED_GRACE_MS;
      const canNotify =
        !missed &&
        "Notification" in window &&
        Notification.permission === "granted";
      if (canNotify) {
        new Notification(reminder.event_title, {
          body: reminderBody(reminder, latest.current.timeFormat),
          tag: reminder.id,
        });
      } else {
        setToasts((current) => [...current, { reminder, missed }]);
      }
      void latest.current.markNotified(reminder.id);
    };

    const now = new Date();
    const timers: number[] = [];
    for (const reminder of reminders) {
      if (reminder.notified_at || reminder.event_is_skipped) continue;
      if (isReminderDue(reminder, now)) {
        deliver(reminder);
        continue;
      }
      const delay = new Date(reminder.remind_at).getTime() - now.getTime();
      if (delay > 0 && delay <= MAX_TIMEOUT_MS) {
        timers.push(window.setTimeout(() => deliver(reminder), delay));
      }
    }
    return () => timers.forEach((timer) => window.clearTimeout(timer));
  }, [reminders, mountedAt]);

  const dismiss = (reminderId: string) =>
    setToasts((current) =>
      current.filter((toast) => toast.reminder.id !== reminderId),
    );

  return { toasts, dismiss };
};

const NO_REMINDERS: Reminder[] = [];

// Delivers the signed-in user's reminders. Call it once, at the app level:
// every call schedules its own notifications.
export const useReminderDelivery = () => {
  const reminders = useQuery(remindersQueryOptions).data ?? NO_REMINDERS;
  const { markNotified } = useReminderMutations();
  const { timeFormat } = usePreferences().preferences;
  return useReminderNotifications(reminders, markNotified, timeFormat);
};
