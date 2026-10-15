import { useEffect, useRef, useState } from "react";
import {
  CALENDAR_GRID_SCALE,
  EVENT_OVERLAP_INSET_PIXELS,
  EVENT_SEPARATOR_PIXELS,
  MINUTES_PER_HOUR,
  formatTime,
} from "../lib/calendar";
import type { TimeFormatPreference } from "../lib/calendar";
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
  continuesFromPreviousDay: boolean;
  continuesToNextDay: boolean;
  timeFormat: TimeFormatPreference;
  onEdit: (entry: TimelineEntry) => void;
  onToggleTask: (entry: TimelineEntry) => void;
};

const CONTINUATION_MARKER_HEIGHT = 6;
const CONTINUATION_MARKER_SPACING = 12;

const ContinuationMarkers = ({
  position,
  count,
  color,
}: {
  position: "top" | "bottom";
  count: number;
  color: string;
}) => (
  <div
    className={`continuation-markers ${position}`}
    style={{ gridTemplateColumns: `repeat(${count}, minmax(0, 1fr))` }}
    aria-hidden="true"
  >
    {Array.from({ length: count }, (_, index) => (
      <i key={index} style={{ backgroundColor: color }} />
    ))}
  </div>
);

export const CalendarEntry = ({
  entry,
  startMinutes,
  durationMinutes,
  trimTop,
  trimBottom,
  overlapDepth,
  continuesFromPreviousDay,
  continuesToNextDay,
  timeFormat,
  onEdit,
  onToggleTask,
}: CalendarEntryProps) => {
  const entryRef = useRef<HTMLElement>(null);
  const [continuationMarkerCount, setContinuationMarkerCount] = useState(1);
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
  const topInset =
    (trimTop ? EVENT_SEPARATOR_PIXELS : 0) +
    (continuesFromPreviousDay ? CONTINUATION_MARKER_HEIGHT : 0);
  const bottomInset =
    (trimBottom ? EVENT_SEPARATOR_PIXELS : 0) +
    (continuesToNextDay ? CONTINUATION_MARKER_HEIGHT : 0);

  useEffect(() => {
    const entryElement = entryRef.current;
    if (!entryElement || (!continuesFromPreviousDay && !continuesToNextDay))
      return;

    const updateMarkerCount = () => {
      const count = Math.max(
        1,
        Math.floor(entryElement.clientWidth / CONTINUATION_MARKER_SPACING),
      );
      setContinuationMarkerCount((currentCount) =>
        currentCount === count ? currentCount : count,
      );
    };

    updateMarkerCount();
    const observer = new ResizeObserver(updateMarkerCount);
    observer.observe(entryElement);
    return () => observer.disconnect();
  }, [continuesFromPreviousDay, continuesToNextDay]);

  return (
    <article
      ref={entryRef}
      className={`grid-entry ${entry.type} ${entry.is_completed ? "completed" : ""} ${hasEarlierOverlap ? "overlapping-later-event" : ""} ${continuesFromPreviousDay ? "continues-from-previous-day" : ""} ${continuesToNextDay ? "continues-to-next-day" : ""}`}
      style={{
        top: `${(startMinutes / MINUTES_PER_HOUR) * CALENDAR_GRID_SCALE}rem`,
        height: `calc(${(durationMinutes / MINUTES_PER_HOUR) * CALENDAR_GRID_SCALE}rem - ${topInset + bottomInset}px)`,
        left: overlapInset,
        right: overlapInset,
        zIndex: hasEarlierOverlap ? 2 : undefined,
        transform: topInset ? `translateY(${topInset}px)` : undefined,
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
      {continuesFromPreviousDay && (
        <ContinuationMarkers
          position="top"
          count={continuationMarkerCount}
          color={mainTag?.color ?? "var(--yellow)"}
        />
      )}
      <strong>{entry.title}</strong>
      <span>
        {entry.type === "event"
          ? [entry.starts_at, entry.ends_at]
              .filter(Boolean)
              .map((time) => formatTime(time, { preference: timeFormat }))
              .join(" – ")
          : "Task"}
      </span>
      {entry.location && <span>{entry.location}</span>}
      {continuesToNextDay && (
        <ContinuationMarkers
          position="bottom"
          count={continuationMarkerCount}
          color={mainTag?.color ?? "var(--yellow)"}
        />
      )}
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
