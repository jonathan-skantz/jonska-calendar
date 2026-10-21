import { formatDate, formatTime } from "../../lib/calendar";
import { formatReminderOffset, groupReminders } from "./reminders";
import type { Reminder } from "../../types";
import { Drawer } from "../../components/Drawer";
import { cx } from "../../lib/classNames";
import { usePreferences } from "../calendar/preferencesContext";

type RemindersDrawerProps = {
  reminders: Reminder[];
  now: Date;
  onOpenEvent: (eventId: string, startsAt: string) => void;
  onRemove: (reminder: Reminder) => void;
  onClose: () => void;
};

// All of the user's reminders in one place, so they can be found and removed
// without remembering which events have them.
export const RemindersDrawer = ({
  reminders,
  now,
  onOpenEvent,
  onRemove,
  onClose,
}: RemindersDrawerProps) => {
  const { timeFormat } = usePreferences().preferences;
  return (
    <Drawer
      eyebrow="Only visible to you"
      title="Reminders"
      className="reminders-drawer"
      onClose={onClose}
    >
      {reminders.length === 0 ? (
        <p className="form-hint">
          No reminders. Add them from an event's details.
        </p>
      ) : (
        groupReminders(reminders, now).map((group) => (
          <section key={group.label} className="reminder-group">
            <h3>{group.label}</h3>
            <ul>
              {group.reminders.map((reminder) => {
                const remindAt = new Date(reminder.remind_at);
                const time = formatTime(remindAt, { preference: timeFormat });
                return (
                  <li
                    key={reminder.id}
                    className={cx(
                      "reminder-row",
                      reminder.event_is_skipped && "skipped",
                    )}
                  >
                    <button
                      type="button"
                      className="reminder-open"
                      onClick={() =>
                        onOpenEvent(reminder.event_id, reminder.event_starts_at)
                      }
                    >
                      <span className="reminder-time">
                        {group.label === "Later" || group.label === "Earlier"
                          ? `${formatDate(remindAt)} ${time}`
                          : time}
                      </span>
                      <strong>{reminder.event_title}</strong>
                      <span className="reminder-meta">
                        {formatReminderOffset(
                          reminder.offset_seconds,
                          reminder.event_is_all_day,
                        )}
                        {reminder.event_is_skipped && " (skipped)"}
                        {reminder.is_shared && " · shared with you"}
                      </span>
                    </button>
                    <button
                      type="button"
                      className="icon-button"
                      aria-label={`Remove reminder for ${reminder.event_title}`}
                      onClick={() => onRemove(reminder)}
                    >
                      ×
                    </button>
                  </li>
                );
              })}
            </ul>
          </section>
        ))
      )}
    </Drawer>
  );
};
