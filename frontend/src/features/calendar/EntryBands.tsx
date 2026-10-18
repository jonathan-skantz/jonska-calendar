import type { CSSProperties, MouseEventHandler } from "react";
import { gridOffset } from "../../lib/calendar";
import { cx } from "../../lib/classNames";
import { LABEL_MIN_MINUTES } from "./entryStyle";
import type { CommuteSegment, GatheringSegment } from "./timelineView";

type BandProps = {
  color: string;
  insets: { left?: string; right?: string };
  onClick: MouseEventHandler<HTMLDivElement>;
};

// Travel time drawn as a hatched band directly above or below the event.
export const CommuteBand = ({
  segment,
  eventTitle,
  // Pixels given up to the separator from an adjacent event.
  trim,
  raised,
  color,
  insets,
  onClick,
}: BandProps & {
  segment: CommuteSegment;
  eventTitle: string;
  trim: number;
  // Lifted above an earlier block it overlaps.
  raised: boolean;
}) => {
  const description = `Commute ${segment.minutes} min ${segment.position} ${eventTitle}`;
  return (
    <div
      className={cx(
        "commute-band",
        segment.position,
        raised && "overlapping-later-event",
      )}
      role="img"
      aria-label={description}
      title={description}
      style={
        {
          "--commute-color": color,
          top: gridOffset(segment.startMinutes),
          height: `calc(${gridOffset(segment.durationMinutes)} - ${trim}px)`,
          ...insets,
          zIndex: raised ? 2 : undefined,
          transform:
            segment.position === "before" && trim
              ? `translateY(${trim}px)`
              : undefined,
        } as CSSProperties
      }
      onClick={onClick}
    >
      {segment.durationMinutes >= LABEL_MIN_MINUTES && (
        <span aria-hidden="true">
          {segment.position === "before" ? "⇢" : "⇠"} {segment.minutes} min
        </span>
      )}
    </div>
  );
};

// Optional gathering may be skipped, so it is a dotted band drawn behind
// other entries instead of part of the event block.
export const OptionalGatheringBand = ({
  segment,
  description,
  chip,
  approximateStart,
  topTrim,
  color,
  insets,
  onClick,
}: BandProps & {
  segment: GatheringSegment;
  description?: string;
  // The visible label, or null when the band is too short for one.
  chip: string | null;
  approximateStart: boolean;
  topTrim: number;
}) => (
  <div
    className={cx(
      "gathering-band optional",
      approximateStart &&
        !segment.continuesFromPreviousDay &&
        "approximate-start",
    )}
    role="img"
    aria-label={description}
    title={description}
    style={
      {
        "--gathering-color": color,
        top: gridOffset(segment.startMinutes),
        height: `calc(${gridOffset(segment.durationMinutes)} - ${topTrim}px)`,
        ...insets,
        transform: topTrim ? `translateY(${topTrim}px)` : undefined,
      } as CSSProperties
    }
    onClick={onClick}
  >
    {chip && (
      <span className="gathering-label" aria-hidden="true">
        {chip}
      </span>
    )}
  </div>
);
