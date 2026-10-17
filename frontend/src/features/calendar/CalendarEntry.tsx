import { useEffect, useRef, useState } from "react";
import type { CSSProperties } from "react";
import {
  EVENT_SEPARATOR_PIXELS,
  formatEventTime,
  gridOffset,
} from "../../lib/calendar";
import type { TimeFormatPreference } from "../../lib/calendar";
import { tagTextColor } from "../../lib/colors";
import {
  getGathering,
  getGatheringStart,
  getMainTag,
  isBusy,
  getOrderedStripTags,
} from "../../lib/timeline";
import type { CommuteSegment, GatheringSegment } from "./timelineView";
import type { Reminder, TimelineEntry } from "../../types";
import { describeReminders } from "../reminders/reminders";
import { cx } from "../../lib/classNames";
import {
  describeApproximateEdges,
  getEntryBackground,
  getHorizontalInsets,
  getTagStripImage,
  getVerticalInsets,
} from "./entryStyle";

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
  // The current user's reminders for this event; shown only as a 🔔 marker.
  reminders?: Reminder[];
  timeFormat: TimeFormatPreference;
  onEdit: (entry: TimelineEntry) => void;
  onToggleTask: (entry: TimelineEntry) => void;
};

const CONTINUATION_MARKER_SPACING = 12;
const COMMUTE_LABEL_MIN_MINUTES = 15;

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
  reminders = [],
  timeFormat,
  onEdit,
  onToggleTask,
}: CalendarEntryProps) => {
  const entryRef = useRef<HTMLElement>(null);
  const [continuationMarkerCount, setContinuationMarkerCount] = useState(1);
  const mainTag = getMainTag(entry);
  const accentColor = mainTag?.color ?? "var(--yellow)";
  const hasEarlierOverlap = overlapDepth > 0;
  const insets = getHorizontalInsets(overlapDepth, overlapsGhost);
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
  const vertical = getVerticalInsets({
    trimTop,
    trimBottom,
    hasCommuteBefore: Boolean(commuteBefore),
    hasCommuteAfter: Boolean(commuteAfter),
    hasOptionalGathering: Boolean(optionalGathering),
    continuesFromPreviousDay,
    continuesToNextDay,
  });
  const startIsApproximate =
    entry.type === "event" && Boolean(entry.start_is_approximate);
  const endIsApproximate =
    entry.type === "event" && Boolean(entry.end_is_approximate);
  // With gathering, the approximate start is when people gather; the event
  // itself still starts exactly. The fade goes on the top-most piece.
  const gathering = getGathering(entry);
  const startTimeIsApproximate = startIsApproximate && gathering.seconds === 0;
  // Fades mark the real start and end, never a midnight where the event
  // continues on another day.
  const articleFadesStart =
    startIsApproximate && !optionalGathering && !continuesFromPreviousDay;
  const articleFadesEnd = endIsApproximate && !continuesToNextDay;
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
  // color: you must be there. A line marks the real start; its offset is
  // measured from the block top, which the separator and continuation insets
  // push down.
  const gatheringHeight = mandatoryGathering
    ? `calc(${gridOffset(mandatoryGathering.startMinutes + mandatoryGathering.durationMinutes - startMinutes)} - ${vertical.top}px)`
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
    const trim = (segment.position === "before" ? trimTop : trimBottom)
      ? EVENT_SEPARATOR_PIXELS
      : 0;
    const description = `Commute ${segment.minutes} min ${segment.position} ${entry.title}`;
    return (
      <div
        key={segment.position}
        className={cx(
          "commute-band",
          segment.position,
          hasEarlierOverlap && "overlapping-later-event",
        )}
        role="img"
        aria-label={description}
        title={description}
        style={
          {
            "--commute-color": accentColor,
            top: gridOffset(segment.startMinutes),
            height: `calc(${gridOffset(segment.durationMinutes)} - ${trim}px)`,
            ...insets,
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
      className={cx(
        "gathering-band optional",
        startIsApproximate &&
          !segment.continuesFromPreviousDay &&
          "approximate-start",
      )}
      role="img"
      aria-label={gatheringDescription}
      title={gatheringDescription}
      style={
        {
          "--gathering-color": accentColor,
          top: gridOffset(segment.startMinutes),
          height: `calc(${gridOffset(segment.durationMinutes)} - ${vertical.outerBandTop}px)`,
          ...insets,
          transform: vertical.outerBandTop
            ? `translateY(${vertical.outerBandTop}px)`
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
          className={cx(
            "grid-entry",
            entry.type,
            entry.is_completed && "completed",
            hasEarlierOverlap && "overlapping-later-event",
            continuesFromPreviousDay && "continues-from-previous-day",
            continuesToNextDay && "continues-to-next-day",
            commuteBefore && "has-commute-before",
            optionalGathering && "has-optional-gathering",
            commuteAfter && "has-commute-after",
            articleFadesStart && "approximate-start",
            isGhost &&
              (entry.is_skipped ? "not-busy skipped" : "not-busy heads-up"),
            articleFadesEnd && "approximate-end",
          )}
          style={
            {
              top: gridOffset(startMinutes),
              height: `calc(${gridOffset(durationMinutes)} - ${vertical.top + vertical.bottom}px)`,
              ...insets,
              zIndex: hasEarlierOverlap && !isGhost ? 2 : undefined,
              transform: vertical.top
                ? `translateY(${vertical.top}px)`
                : undefined,
              ...(entry.type === "event" && mainTag
                ? {
                    ...(isGhost
                      ? { "--tag-color": mainTag.color }
                      : {
                          backgroundColor: mainTag.color,
                          color: tagTextColor(mainTag.color),
                        }),
                    ...getEntryBackground(
                      getTagStripImage(
                        mainTag,
                        getOrderedStripTags(entry).filter(
                          (tag) => tag.id !== mainTag.id,
                        ),
                      ),
                      articleFadesStart,
                      articleFadesEnd,
                    ),
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
              color={accentColor}
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
            <i
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
          <span
            title={describeApproximateEdges(
              startIsApproximate,
              endIsApproximate,
            )}
          >
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
              color={accentColor}
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
