import { tagTextColor } from "../../lib/colors";
import type { CSSProperties } from "react";
import { getMainTag, isBusy } from "../../lib/timeline";
import type { Reminder, TimelineEntry } from "../../types";
import { describeReminders } from "../reminders/reminders";
import { cx } from "../../lib/classNames";
import { describeEntryRecurrence } from "../../lib/recurrence";

export function AllDayEvent({
  entry,
  startIndex,
  endIndex,
  lane,
  continuesFromPreviousWeek,
  continuesToNextWeek,
  onEdit,
  reminders = [],
}: {
  entry: TimelineEntry;
  startIndex: number;
  endIndex: number;
  lane: number;
  continuesFromPreviousWeek: boolean;
  continuesToNextWeek: boolean;
  onEdit: (entry: TimelineEntry) => void;
  reminders?: Reminder[];
}) {
  const tag = getMainTag(entry);
  const backgroundColor = tag?.color ?? "var(--moss-dark)";
  // Heads-up and skipped all-day events (birthdays, holidays) are outlined.
  const isGhost = !isBusy(entry);
  return (
    <div
      className="all-day-event-shell"
      style={{
        gridColumn: `${startIndex + 1} / ${endIndex + 2}`,
        gridRow: lane + 1,
      }}
    >
      {continuesFromPreviousWeek && (
        <span
          className="all-day-triangle left"
          style={{ backgroundColor }}
          aria-hidden="true"
        />
      )}
      <button
        type="button"
        className={cx(
          "all-day-event",
          continuesToNextWeek && "continues-to-next-week",
          isGhost && "not-busy",
        )}
        style={
          isGhost
            ? ({ "--tag-color": backgroundColor } as CSSProperties)
            : {
                backgroundColor,
                ...(tag ? { color: tagTextColor(tag.color) } : {}),
              }
        }
        onClick={(event) => {
          event.stopPropagation();
          onEdit(entry);
        }}
        title={entry.title}
      >
        <span
          className={cx(
            "all-day-title",
            continuesToNextWeek && "continues-to-next-week",
          )}
        >
          {reminders.length > 0 && (
            <i
              className="reminder-marker"
              role="img"
              aria-label={describeReminders(reminders)}
              title={describeReminders(reminders)}
            >
              🔔{" "}
            </i>
          )}
          {entry.recurrence && (
            <i
              className="repeat-marker"
              role="img"
              aria-label={`Repeats: ${describeEntryRecurrence(entry)}`}
              title={describeEntryRecurrence(entry)}
            >
              🔁{" "}
            </i>
          )}
          {entry.title}
        </span>
      </button>
    </div>
  );
}
