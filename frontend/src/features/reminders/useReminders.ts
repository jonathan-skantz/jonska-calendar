import { useQuery, useQueryClient } from "@tanstack/react-query";
import { api } from "../../lib/api";
import { useActionMutation } from "../../hooks/useActionMutation";
import { invalidateReminders } from "../../lib/queryInvalidation";
import { queryKeys } from "../../lib/queryKeys";
import type { Reminder } from "../../types";

// Refetched periodically so reminders set on another device get scheduled.
const REMINDER_REFETCH_MS = 5 * 60 * 1000;

export const useReminders = (enabled: boolean) =>
  useQuery({
    queryKey: queryKeys.reminders,
    queryFn: () => api<Reminder[]>("/api/reminders"),
    enabled,
    refetchInterval: REMINDER_REFETCH_MS,
  });

// Notification permission must be requested from a user gesture, so it is
// asked for when the user adds their first reminder.
export const requestNotificationPermission = () => {
  if ("Notification" in window && Notification.permission === "default") {
    void Notification.requestPermission();
  }
};

// add and remove resolve to an error message, or null when they succeeded.
export const useReminderMutations = () => {
  const queryClient = useQueryClient();
  const run = useActionMutation("Unable to update reminders", () =>
    invalidateReminders(queryClient),
  );
  const send = (path: string, method: string, body?: unknown) =>
    run({ path, method, body });

  return {
    add: (eventId: string, offsetSeconds: number) => {
      requestNotificationPermission();
      return send(`/api/events/${eventId}/reminders`, "POST", {
        offset_seconds: offsetSeconds,
      });
    },
    remove: (reminderId: string) =>
      send(`/api/reminders/${reminderId}`, "DELETE"),
    markNotified: (reminderId: string) =>
      send(`/api/reminders/${reminderId}/notified`, "POST"),
  };
};
