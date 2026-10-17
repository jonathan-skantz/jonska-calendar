import { useState } from "react";
import type { TimeFormatPreference } from "../../lib/calendar";
import type { Reminder } from "../../types";
import type { ReminderItem } from "./ReminderEditor";
import { getRemindersByEvent } from "./reminders";
import { useReminderMutations, useReminders } from "./useReminders";
import { useReminderNotifications } from "./useReminderNotifications";

const NO_REMINDERS: Reminder[] = [];

// The current user's reminders, their delivery while the app is open, and the
// add/remove actions the event views share. The last action's error is kept
// for whichever view is showing reminders.
export const useEventReminders = (
  enabled: boolean,
  timeFormat: TimeFormatPreference,
) => {
  const remindersQuery = useReminders(enabled);
  const reminders = remindersQuery.data ?? NO_REMINDERS;
  const remindersByEvent = getRemindersByEvent(reminders);
  const mutations = useReminderMutations();
  const notifications = useReminderNotifications(
    reminders,
    mutations.markNotified,
    timeFormat,
  );
  const [error, setError] = useState<string | null>(null);

  return {
    reminders,
    remindersByEvent,
    notifications,
    error,
    clearError: () => setError(null),
    itemsFor: (eventId: string): ReminderItem[] =>
      (remindersByEvent.get(eventId) ?? []).map((reminder) => ({
        id: reminder.id,
        offsetSeconds: reminder.offset_seconds,
      })),
    add: async (eventId: string, offsetSeconds: number) =>
      setError(await mutations.add(eventId, offsetSeconds)),
    remove: async (reminderId: string) =>
      setError(await mutations.remove(reminderId)),
  };
};
