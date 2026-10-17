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
import {
  getGathering,
  getGatheringStart,
  getMainTag,
  isBusy,
  getOrderedStripTags,
} from "../lib/timeline";
import type {
  CommuteSegment,
  GatheringSegment,
} from "../features/calendar/timelineView";
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
  gatheringSegment?: GatheringSegment | null;
  showBody: boolean;
  // A busy block over a not-busy ghost is inset so the ghost's edge shows.
  overlapsGhost?: boolean;
  timeFormat: TimeFormatPreference;
  onEdit: (entry: TimelineEntry) => void;
  onToggleTask: (entry: TimelineEntry) => void;
};

const CONTINUATION_MARKER_HEIGHT = 6;
const CONTINUATION_MARKER_SPACING = 12;
const COMMUTE_LABEL_MIN_MINUTES = 15;
const GHOST_EDGE_PIXELS = 8;

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
  gatheringSegment,
  showBody,
  overlapsGhost = false,
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
  const leftInset =
    overlapInset ?? (overlapsGhost ? `${3 + GHOST_EDGE_PIXELS}px` : undefined);
  // Heads-up and skipped events don't take up time: an outlined ghost that
  // busy blocks draw over, marked with why it is not busy.
  const isGhost = entry.type === "event" && !isBusy(entry);
  const ghostMarker = !isGhost
    ? null
    : entry.is_skipped
      ? { icon: "⊘", label: "Skipping" }
      : { icon: "ⓘ", label: "Heads-up" };
  const commuteBefore = commuteSegments.find(
    (segment) => segment.position === "before",
  );
  const commuteAfter = commuteSegments.find(
    (segment) => segment.position === "after",
  );
  const mandatoryGathering =
    gatheringSegment && !gatheringSegment.isOptional ? gatheringSegment : null;
  const optionalGathering = gatheringSegment?.isOptional
    ? gatheringSegment
    : null;
  // The separator from an adjacent event belongs to the outermost block,
  // which is the commute band or optional gathering band when there is one.
  const outerBandTrim = trimTop && !commuteBefore ? EVENT_SEPARATOR_PIXELS : 0;
  const topInset =
    (trimTop && !commuteBefore && !optionalGathering
      ? EVENT_SEPARATOR_PIXELS
      : 0) + (continuesFromPreviousDay ? CONTINUATION_MARKER_HEIGHT : 0);
  const bottomInset =
    (trimBottom && !commuteAfter ? EVENT_SEPARATOR_PIXELS : 0) +
    (continuesToNextDay ? CONTINUATION_MARKER_HEIGHT : 0);
  const commuteColor = mainTag?.color ?? "var(--yellow)";
  const startIsApproximate =
    entry.type === "event" && Boolean(entry.start_is_approximate);
  const endIsApproximate =
    entry.type === "event" && Boolean(entry.end_is_approximate);
  // With gathering, the approximate start is when people gather; the event
  // itself still starts exactly. The fade goes on the top-most piece.
  const gathering = getGathering(entry);
  const startTimeIsApproximate = startIsApproximate && gathering.seconds === 0;
  const articleFadesStart = startIsApproximate && !optionalGathering;
  const gatheringTime = formatEventTime(
    getGatheringStart(entry),
    startIsApproximate,
    { preference: timeFormat },
  );
  const gatheringChip = `${gathering.label ?? "Gather"} ${gatheringTime}`;
  const gatheringDescription = gatheringSegment
    ? gatheringSegment.isOptional
      ? `Optional gathering${gathering.label ? `: ${gathering.label}` : ""} from ${gatheringTime}`
      : `${gathering.label ?? "Gathering"} from ${gatheringTime}, before ${entry.title} starts`
    : undefined;
  const showGatheringChip = Boolean(
    gatheringSegment &&
    gatheringSegment.durationMinutes >= COMMUTE_LABEL_MIN_MINUTES,
  );
  // Mandatory gathering is the top part of the event block, in full event
  // color: you must be there. Its height is measured from the block top,
  // which the separator and continuation insets push down.
  const gatheringHeight = mandatoryGathering
    ? `calc(${gridOffset(mandatoryGathering.startMinutes + mandatoryGathering.durationMinutes - startMinutes)} - ${topInset}px)`
    : undefined;
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
    ...(articleFadesStart
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
            left: leftInset,
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

  // Optional gathering may be skipped, so it is a dotted band drawn behind
  // other entries instead of part of the event block.
  const renderOptionalGathering = (segment: GatheringSegment) => (
    <div
      className={`gathering-band optional ${startIsApproximate ? "approximate-start" : ""}`}
      role="img"
      aria-label={gatheringDescription}
      title={gatheringDescription}
      style={
        {
          "--gathering-color": mainTag?.color ?? "var(--yellow)",
          top: gridOffset(segment.startMinutes),
          height: `calc(${gridOffset(segment.durationMinutes)} - ${outerBandTrim}px)`,
          left: leftInset,
          right: overlapInset,
          transform: outerBandTrim
            ? `translateY(${outerBandTrim}px)`
            : undefined,
        } as CSSProperties
      }
      onClick={(event) => {
        event.stopPropagation();
        openEditor();
      }}
    >
      {showGatheringChip && (
        <span className="gathering-label" aria-hidden="true">
          {gatheringChip}
        </span>
      )}
    </div>
  );

  return (
    <>
      {commuteBefore && renderCommuteBand(commuteBefore)}
      {optionalGathering && renderOptionalGathering(optionalGathering)}
      {showBody && (
        <article
          ref={entryRef}
          className={`grid-entry ${entry.type} ${entry.is_completed ? "completed" : ""} ${hasEarlierOverlap ? "overlapping-later-event" : ""} ${continuesFromPreviousDay ? "continues-from-previous-day" : ""} ${continuesToNextDay ? "continues-to-next-day" : ""} ${commuteBefore ? "has-commute-before" : ""} ${optionalGathering ? "has-optional-gathering" : ""} ${commuteAfter ? "has-commute-after" : ""} ${articleFadesStart ? "approximate-start" : ""} ${isGhost ? `not-busy ${entry.is_skipped ? "skipped" : "heads-up"}` : ""} ${endIsApproximate ? "approximate-end" : ""}`}
          style={
            {
              top: gridOffset(startMinutes),
              height: `calc(${gridOffset(durationMinutes)} - ${topInset + bottomInset}px)`,
              left: leftInset,
              right: overlapInset,
              zIndex: hasEarlierOverlap && !isGhost ? 2 : undefined,
              transform: topInset ? `translateY(${topInset}px)` : undefined,
              paddingTop: gatheringHeight
                ? `calc(${gatheringHeight} + 4px)`
                : undefined,
              ...(entry.type === "event" && mainTag
                ? {
                    ...(isGhost
                      ? { "--tag-color": mainTag.color }
                      : {
                          backgroundColor: mainTag.color,
                          color: tagTextColor(mainTag.color),
                        }),
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
            } as CSSProperties
          }
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
          {mandatoryGathering && showGatheringChip && (
            <span
              className="gathering-label"
              title={gatheringDescription}
              aria-label={gatheringDescription}
            >
              {gatheringChip}
            </span>
          )}
          {mandatoryGathering?.endsAtEventStart && (
            <span
              className="gathering-start"
              style={{ top: gatheringHeight }}
              role={showGatheringChip ? undefined : "img"}
              aria-hidden={showGatheringChip || undefined}
              aria-label={showGatheringChip ? undefined : gatheringDescription}
              title={gatheringDescription}
            />
          )}
          <strong>
            {ghostMarker && (
              <i
                className="ghost-marker"
                role="img"
                aria-label={ghostMarker.label}
                title={ghostMarker.label}
              >
                {ghostMarker.icon}{" "}
              </i>
            )}
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
            {entry.is_skipped && "Skipping · "}
            {entry.type === "event"
              ? [
                  formatEventTime(entry.starts_at, startTimeIsApproximate, {
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
