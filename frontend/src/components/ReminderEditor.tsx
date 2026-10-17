import { useState } from "react";
import {
  ALL_DAY_REMINDER_PRESETS,
  MAX_REMINDER_OFFSET_SECONDS,
  TIMED_REMINDER_PRESETS,
  formatReminderOffset,
} from "../features/reminders/reminders";

export type ReminderItem = { id: string; offsetSeconds: number };

type ReminderEditorProps = {
  reminders: ReminderItem[];
  isAllDay: boolean;
  error?: string | null;
  onAdd: (offsetSeconds: number) => void;
  onRemove: (reminder: ReminderItem) => void;
};

// Lists the current user's reminders for one event. The caller decides
// whether a change is saved right away (saved events) or queued until the
// event is created.
export const ReminderEditor = ({
  reminders,
  isAllDay,
  error,
  onAdd,
  onRemove,
}: ReminderEditorProps) => {
  const [customMinutes, setCustomMinutes] = useState("");
  const taken = new Set(reminders.map((reminder) => reminder.offsetSeconds));
  const presets = (
    isAllDay ? ALL_DAY_REMINDER_PRESETS : TIMED_REMINDER_PRESETS
  ).filter((offset) => !taken.has(offset));
  const customSeconds = Math.round(Number(customMinutes)) * 60;
  const customIsValid =
    customMinutes !== "" &&
    customSeconds >= 0 &&
    customSeconds <= MAX_REMINDER_OFFSET_SECONDS &&
    !taken.has(customSeconds);

  return (
    <div className="reminder-editor" role="group" aria-label="Reminders">
      <span className="reminder-editor-label">Reminders</span>
      {reminders.length > 0 && (
        <ul className="reminder-chips">
          {[...reminders]
            .sort((left, right) => left.offsetSeconds - right.offsetSeconds)
            .map((reminder) => {
              const label = formatReminderOffset(
                reminder.offsetSeconds,
                isAllDay,
              );
              return (
                <li key={reminder.id} className="reminder-chip">
                  {label}
                  <button
                    type="button"
                    aria-label={`Remove reminder ${label}`}
                    onClick={() => onRemove(reminder)}
                  >
                    ×
                  </button>
                </li>
              );
            })}
        </ul>
      )}
      <div className="reminder-add">
        {presets.length > 0 && (
          <select
            aria-label="Add reminder"
            value=""
            onChange={(event) => {
              if (event.target.value) onAdd(Number(event.target.value));
            }}
          >
            <option value="">Add reminder…</option>
            {presets.map((offset) => (
              <option key={offset} value={offset}>
                {formatReminderOffset(offset, isAllDay)}
              </option>
            ))}
          </select>
        )}
        <input
          type="number"
          inputMode="numeric"
          min={0}
          max={MAX_REMINDER_OFFSET_SECONDS / 60}
          step={1}
          placeholder="min before"
          aria-label="Custom reminder (minutes before)"
          value={customMinutes}
          onChange={(event) => setCustomMinutes(event.target.value)}
        />
        <button
          type="button"
          className="text-button"
          disabled={!customIsValid}
          onClick={() => {
            onAdd(customSeconds);
            setCustomMinutes("");
          }}
        >
          Add
        </button>
      </div>
      {error && <p className="error">{error}</p>}
    </div>
  );
};
