import { useEffect, useRef, useState } from "react";
import type { CSSProperties } from "react";
import {
  CALENDAR_GRID_SCALE,
  EVENT_OVERLAP_INSET_PIXELS,
  EVENT_SEPARATOR_PIXELS,
  MINUTES_PER_HOUR,
  formatEventTime,
} from "../lib/calendar";
import type { TimeFormatPreference } from "../lib/calendar";
import { tagTextColor } from "../lib/colors";
import { getMainTag, getOrderedStripTags } from "../lib/timeline";
import type { CommuteSegment } from "../features/calendar/timelineView";
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
  commuteSegments: CommuteSegment[];
  showBody: boolean;
  timeFormat: TimeFormatPreference;
  onEdit: (entry: TimelineEntry) => void;
  onToggleTask: (entry: TimelineEntry) => void;
};

const CONTINUATION_MARKER_HEIGHT = 6;
const CONTINUATION_MARKER_SPACING = 12;
const COMMUTE_LABEL_MIN_MINUTES = 15;

const gridOffset = (minutes: number) =>
  `${(minutes / MINUTES_PER_HOUR) * CALENDAR_GRID_SCALE}rem`;

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
  commuteSegments,
  showBody,
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
  const commuteBefore = commuteSegments.find(
    (segment) => segment.position === "before",
  );
  const commuteAfter = commuteSegments.find(
    (segment) => segment.position === "after",
  );
  // The separator from an adjacent event belongs to the outermost block,
  // which is the commute band when there is one.
  const topInset =
    (trimTop && !commuteBefore ? EVENT_SEPARATOR_PIXELS : 0) +
    (continuesFromPreviousDay ? CONTINUATION_MARKER_HEIGHT : 0);
  const bottomInset =
    (trimBottom && !commuteAfter ? EVENT_SEPARATOR_PIXELS : 0) +
    (continuesToNextDay ? CONTINUATION_MARKER_HEIGHT : 0);
  const commuteColor = mainTag?.color ?? "var(--yellow)";
  const startIsApproximate =
    entry.type === "event" && Boolean(entry.start_is_approximate);
  const endIsApproximate =
    entry.type === "event" && Boolean(entry.end_is_approximate);
  // Approximate edges fade into the paper color. The tag strip is the first
  // layer so it stays crisp above the fade.
  const backgroundLayers = [
    ...(stripStops.length
      ? [
          {
            image: `linear-gradient(to right, ${stripStops.join(", ")})`,
            size: "100% 3px",
            position: "top",
          },
        ]
      : []),
    ...(startIsApproximate
      ? [
          {
            image: "linear-gradient(to bottom, var(--paper), transparent)",
            size: "100% var(--approximate-fade)",
            position: "top",
          },
        ]
      : []),
    ...(endIsApproximate
      ? [
          {
            image: "linear-gradient(to top, var(--paper), transparent)",
            size: "100% var(--approximate-fade)",
            position: "bottom",
          },
        ]
      : []),
  ];
  const approximateDescription =
    startIsApproximate && endIsApproximate
      ? "Approximate start and end"
      : startIsApproximate
        ? "Approximate start"
        : endIsApproximate
          ? "Approximate end"
          : undefined;
  // Shared entries open a read-only view; App decides which drawer to show.
  const openEditor = () => onEdit(entry);

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

  const renderCommuteBand = (segment: CommuteSegment) => {
    const trim =
      segment.position === "before"
        ? trimTop
          ? EVENT_SEPARATOR_PIXELS
          : 0
        : trimBottom
          ? EVENT_SEPARATOR_PIXELS
          : 0;
    const description = `Commute ${segment.minutes} min ${segment.position} ${entry.title}`;
    return (
      <div
        key={segment.position}
        className={`commute-band ${segment.position} ${hasEarlierOverlap ? "overlapping-later-event" : ""}`}
        role="img"
        aria-label={description}
        title={description}
        style={
          {
            "--commute-color": commuteColor,
            top: gridOffset(segment.startMinutes),
            height: `calc(${gridOffset(segment.durationMinutes)} - ${trim}px)`,
            left: overlapInset,
            right: overlapInset,
            zIndex: hasEarlierOverlap ? 2 : undefined,
            transform:
              segment.position === "before" && trim
                ? `translateY(${trim}px)`
                : undefined,
          } as CSSProperties
        }
        onClick={(event) => {
          event.stopPropagation();
          openEditor();
        }}
      >
        {segment.durationMinutes >= COMMUTE_LABEL_MIN_MINUTES && (
          <span aria-hidden="true">
            {segment.position === "before" ? "⇢" : "⇠"} {segment.minutes} min
          </span>
        )}
      </div>
    );
  };

  return (
    <>
      {commuteBefore && renderCommuteBand(commuteBefore)}
      {showBody && (
        <article
          ref={entryRef}
          className={`grid-entry ${entry.type} ${entry.is_completed ? "completed" : ""} ${hasEarlierOverlap ? "overlapping-later-event" : ""} ${continuesFromPreviousDay ? "continues-from-previous-day" : ""} ${continuesToNextDay ? "continues-to-next-day" : ""} ${commuteBefore ? "has-commute-before" : ""} ${commuteAfter ? "has-commute-after" : ""} ${startIsApproximate ? "approximate-start" : ""} ${endIsApproximate ? "approximate-end" : ""}`}
          style={{
            top: gridOffset(startMinutes),
            height: `calc(${gridOffset(durationMinutes)} - ${topInset + bottomInset}px)`,
            left: overlapInset,
            right: overlapInset,
            zIndex: hasEarlierOverlap ? 2 : undefined,
            transform: topInset ? `translateY(${topInset}px)` : undefined,
            ...(entry.type === "event" && mainTag
              ? {
                  backgroundColor: mainTag.color,
                  color: tagTextColor(mainTag.color),
                  backgroundImage: backgroundLayers
                    .map((layer) => layer.image)
                    .join(", "),
                  backgroundSize: backgroundLayers
                    .map((layer) => layer.size)
                    .join(", "),
                  backgroundPosition: backgroundLayers
                    .map((layer) => layer.position)
                    .join(", "),
                  backgroundRepeat: "no-repeat",
                }
              : {}),
          }}
          onClick={(event) => {
            event.stopPropagation();
            openEditor();
          }}
        >
          {continuesFromPreviousDay && (
            <ContinuationMarkers
              position="top"
              count={continuationMarkerCount}
              color={mainTag?.color ?? "var(--yellow)"}
            />
          )}
          <strong>
            {entry.share_count ? (
              <i
                className="share-marker"
                title={`Shared with ${entry.share_count} ${entry.share_count === 1 ? "person" : "people"}`}
              >
                ⇄{" "}
              </i>
            ) : null}
            {entry.title}
          </strong>
          <span title={approximateDescription}>
            {entry.type === "event"
              ? [
                  formatEventTime(entry.starts_at, startIsApproximate, {
                    preference: timeFormat,
                  }),
                  formatEventTime(entry.ends_at, endIsApproximate, {
                    preference: timeFormat,
                  }),
                ]
                  .filter(Boolean)
                  .join(" – ")
              : "Task"}
          </span>
          {entry.location && <span>{entry.location}</span>}
          {entry.shared_by && <span>from {entry.shared_by}</span>}
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
      )}
      {commuteAfter && renderCommuteBand(commuteAfter)}
    </>
  );
};
