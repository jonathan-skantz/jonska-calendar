import { formatTime } from "../lib/calendar";
import type { TimeFormatPreference } from "../lib/calendar";
import type { ReminderToast } from "../features/reminders/useReminderNotifications";

type ReminderToastsProps = {
  toasts: ReminderToast[];
  timeFormat: TimeFormatPreference;
  onOpenEvent: (eventId: string) => void;
  onDismiss: (reminderId: string) => void;
};

export const ReminderToasts = ({
  toasts,
  timeFormat,
  onOpenEvent,
  onDismiss,
}: ReminderToastsProps) =>
  toasts.length === 0 ? null : (
    <div className="reminder-toasts" role="status" aria-live="polite">
      {toasts.map(({ reminder, missed }) => (
        <div key={reminder.id} className="reminder-toast">
          <button
            type="button"
            className="reminder-open"
            onClick={() => {
              onOpenEvent(reminder.event_id);
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
