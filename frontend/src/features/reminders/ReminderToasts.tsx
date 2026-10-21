import { formatTime } from "../../lib/calendar";
import type { ReminderToast } from "./useReminderNotifications";
import { usePreferences } from "../calendar/preferencesContext";

type ReminderToastsProps = {
  toasts: ReminderToast[];
  onOpenEvent: (eventId: string, startsAt: string) => void;
  onDismiss: (reminderId: string) => void;
};

export const ReminderToasts = ({
  toasts,
  onOpenEvent,
  onDismiss,
}: ReminderToastsProps) => {
  const { timeFormat } = usePreferences().preferences;
  return toasts.length === 0 ? null : (
    <div className="reminder-toasts" role="status" aria-live="polite">
      {toasts.map(({ reminder, missed }) => (
        <div key={reminder.id} className="reminder-toast">
          <button
            type="button"
            className="reminder-open"
            onClick={() => {
              onOpenEvent(reminder.event_id, reminder.event_starts_at);
              onDismiss(reminder.id);
            }}
          >
            <span className="eyebrow">
              {missed ? "Missed reminder" : "Reminder"}
            </span>
            <strong>{reminder.event_title}</strong>
            <span>
              {reminder.event_is_all_day
                ? "All day"
                : formatTime(reminder.event_starts_at, {
                    preference: timeFormat,
                  })}
              {reminder.event_location && ` · ${reminder.event_location}`}
            </span>
          </button>
          <button
            type="button"
            className="icon-button"
            aria-label={`Dismiss reminder for ${reminder.event_title}`}
            onClick={() => onDismiss(reminder.id)}
          >
            ×
          </button>
        </div>
      ))}
    </div>
  );
};
