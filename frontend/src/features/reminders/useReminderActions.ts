import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import type { Reminder } from "../../types";
import type { ReminderItem } from "./ReminderEditor";
import { firstOfEachReminder, getRemindersByEvent } from "./reminders";
import { remindersQueryOptions, useReminderMutations } from "./useReminders";

const indexReminders = (list: Reminder[]) => ({
  list: firstOfEachReminder(list),
  byEvent: getRemindersByEvent(list),
});
const NO_REMINDERS = indexReminders([]);

// The current user's reminders and the add/remove actions the event views
// share. Safe to call from any component; each caller keeps its own error.
export const useReminderActions = () => {
  // A module-level select only re-runs when the reminders change.
  const reminders =
    useQuery({ ...remindersQueryOptions, select: indexReminders }).data ??
    NO_REMINDERS;
  const mutations = useReminderMutations();
  const [error, setError] = useState<string | null>(null);

  return {
    reminders: reminders.list,
    remindersByEvent: reminders.byEvent,
    error,
    clearError: () => setError(null),
    itemsFor: (eventId: string): ReminderItem[] =>
      (reminders.byEvent.get(eventId) ?? []).map((reminder) => ({
        id: reminder.id,
        offsetSeconds: reminder.offset_seconds,
      })),
    add: async (eventId: string, offsetSeconds: number) =>
      setError(await mutations.add(eventId, offsetSeconds)),
    remove: async (reminderId: string) =>
      setError(await mutations.remove(reminderId)),
  };
};
