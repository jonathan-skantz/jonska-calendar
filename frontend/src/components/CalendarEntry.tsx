import {
  CALENDAR_GRID_SCALE,
  EVENT_OVERLAP_INSET_PIXELS,
  EVENT_SEPARATOR_PIXELS,
  MINUTES_PER_HOUR,
  formatTime,
} from "../lib/calendar";
import { tagTextColor } from "../lib/colors";
import { getMainTag, getOrderedStripTags } from "../lib/timeline";
import type { TimelineEntry } from "../types";

type CalendarEntryProps = {
  entry: TimelineEntry;
  startMinutes: number;
  durationMinutes: number;
  trimTop: boolean;
  trimBottom: boolean;
  overlapDepth: number;
  onEdit: (entry: TimelineEntry) => void;
  onToggleTask: (entry: TimelineEntry) => void;
};

export const CalendarEntry = ({
  entry,
  startMinutes,
  durationMinutes,
  trimTop,
  trimBottom,
  overlapDepth,
  onEdit,
  onToggleTask,
}: CalendarEntryProps) => {
  const mainTag = getMainTag(entry);
  const stripTags = getOrderedStripTags(entry).filter(
    (tag) => tag.id !== mainTag?.id,
  );
  const additionalStripWidth = stripTags.length ? 50 / stripTags.length : 0;
  const stripStops = stripTags.length
    ? [
        `${mainTag?.color} 0%, ${mainTag?.color} 50%`,
        ...stripTags.map(
          (tag, index) =>
            `${tag.color} ${50 + index * additionalStripWidth}%, ${tag.color} ${50 + (index + 1) * additionalStripWidth}%`,
        ),
      ]
    : mainTag
      ? [`${mainTag.color} 0%, ${mainTag.color} 100%`]
      : [];
  const hasEarlierOverlap = overlapDepth > 0;
  const overlapInset = hasEarlierOverlap
    ? `${3 + overlapDepth * EVENT_OVERLAP_INSET_PIXELS}px`
    : undefined;

  return (
    <article
      className={`grid-entry ${entry.type} ${entry.is_completed ? "completed" : ""} ${hasEarlierOverlap ? "overlapping-later-event" : ""}`}
      style={{
        top: `${(startMinutes / MINUTES_PER_HOUR) * CALENDAR_GRID_SCALE}rem`,
        height: `calc(${(durationMinutes / MINUTES_PER_HOUR) * CALENDAR_GRID_SCALE}rem - ${(trimTop ? EVENT_SEPARATOR_PIXELS : 0) + (trimBottom ? EVENT_SEPARATOR_PIXELS : 0)}px)`,
        left: overlapInset,
        right: overlapInset,
        zIndex: hasEarlierOverlap ? 2 : undefined,
        transform: trimTop ? `translateY(${EVENT_SEPARATOR_PIXELS}px)` : undefined,
        ...(entry.type === "event" && mainTag
          ? {
              backgroundColor: mainTag.color,
              color: tagTextColor(mainTag.color),
              backgroundImage: `linear-gradient(to right, ${stripStops.join(", ")})`,
              backgroundSize: "100% 3px",
              backgroundPosition: "top",
              backgroundRepeat: "no-repeat",
            }
          : {}),
      }}
      onClick={(event) => {
        event.stopPropagation();
        if (!entry.is_shared) onEdit(entry);
      }}
    >
      <strong>{entry.title}</strong>
      <span>
        {entry.type === "event"
          ? `${formatTime(entry.starts_at)}${entry.ends_at ? ` – ${formatTime(entry.ends_at)}` : ""}`
          : "Task"}
      </span>
      {entry.location && <span>{entry.location}</span>}
      {entry.type === "task" && (
        <input
          type="checkbox"
          checked={entry.is_completed}
          onClick={(event) => event.stopPropagation()}
          onChange={() => onToggleTask(entry)}
          aria-label={`Complete ${entry.title}`}
        />
      )}
    </article>
  );
};
