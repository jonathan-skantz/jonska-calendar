import { tagTextColor } from "../lib/colors";
import { getMainTag } from "../lib/timeline";
import type { TimelineEntry } from "../types";

export function AllDayEvent({
  entry,
  startIndex,
  endIndex,
  lane,
  continuesFromPreviousWeek,
  continuesToNextWeek,
  onEdit,
}: {
  entry: TimelineEntry;
  startIndex: number;
  endIndex: number;
  lane: number;
  continuesFromPreviousWeek: boolean;
  continuesToNextWeek: boolean;
  onEdit: (entry: TimelineEntry) => void;
}) {
  const tag = getMainTag(entry);
  const backgroundColor = tag?.color ?? "var(--moss-dark)";
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
        className={`all-day-event ${continuesToNextWeek ? "continues-to-next-week" : ""}`}
        style={{
          backgroundColor,
          ...(tag ? { color: tagTextColor(tag.color) } : {}),
        }}
        onClick={(event) => {
          event.stopPropagation();
          onEdit(entry);
        }}
        title={entry.title}
      >
        <span
          className={`all-day-title ${continuesToNextWeek ? "continues-to-next-week" : ""}`}
        >
          {entry.title}
        </span>
      </button>
    </div>
  );
}
