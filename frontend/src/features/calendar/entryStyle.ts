import { EVENT_SEPARATOR_PIXELS } from "../../lib/calendar";
import type { Tag } from "../../types";
import type { HorizontalPlacement } from "./timelineView";

// Space kept free at each edge of a block, so side-by-side blocks have a gap.
const BASE_INSET_PIXELS = 3;
// A busy block over a not-busy ghost leaves this much of the ghost showing.
const GHOST_EDGE_PIXELS = 8;
const CONTINUATION_MARKER_HEIGHT = 6;
// Bands and chips shorter than this have no room for a text label.
export const LABEL_MIN_MINUTES = 15;

// The tag strip along the top of an event: the main tag fills the left half
// and any other tags share the right half, or the main tag fills it all.
export const getTagStripImage = (
  mainTag: Tag | undefined,
  otherTags: Tag[],
) => {
  if (!mainTag) return null;
  if (otherTags.length === 0) {
    return `linear-gradient(to right, ${mainTag.color} 0%, ${mainTag.color} 100%)`;
  }
  const width = 50 / otherTags.length;
  const stops = [
    `${mainTag.color} 0%, ${mainTag.color} 50%`,
    ...otherTags.map(
      (tag, index) =>
        `${tag.color} ${50 + index * width}%, ${tag.color} ${50 + (index + 1) * width}%`,
    ),
  ];
  return `linear-gradient(to right, ${stops.join(", ")})`;
};

// Approximate edges fade into the paper color. The tag strip is the first
// layer so it stays crisp above the fade.
export const getEntryBackground = (
  stripImage: string | null,
  fadesStart: boolean,
  fadesEnd: boolean,
) => {
  const layers: { image: string; size: string; position: string }[] = [];
  if (stripImage) {
    layers.push({ image: stripImage, size: "100% 3px", position: "top" });
  }
  if (fadesStart) {
    layers.push({
      image: "linear-gradient(to bottom, var(--paper), transparent)",
      size: "100% var(--approximate-fade)",
      position: "top",
    });
  }
  if (fadesEnd) {
    layers.push({
      image: "linear-gradient(to top, var(--paper), transparent)",
      size: "100% var(--approximate-fade)",
      position: "bottom",
    });
  }
  return {
    backgroundImage: layers.map((layer) => layer.image).join(", "),
    backgroundSize: layers.map((layer) => layer.size).join(", "),
    backgroundPosition: layers.map((layer) => layer.position).join(", "),
    backgroundRepeat: "no-repeat",
  };
};

const edgeOffset = (fraction: number) =>
  `calc(${+(fraction * 100).toFixed(3)}% + ${BASE_INSET_PIXELS}px)`;

// A block placed beside or over others gets its own share of the column; a
// full-width block over a ghost steps in on the left so the ghost shows.
export const getHorizontalInsets = (
  { left, right }: HorizontalPlacement,
  overlapsGhost: boolean,
) => ({
  left:
    left > 0
      ? edgeOffset(left)
      : overlapsGhost
        ? `${BASE_INSET_PIXELS + GHOST_EDGE_PIXELS}px`
        : undefined,
  right: right < 1 ? edgeOffset(1 - right) : undefined,
});

type VerticalInsetOptions = {
  trimTop: boolean;
  trimBottom: boolean;
  hasCommuteBefore: boolean;
  hasCommuteAfter: boolean;
  hasOptionalGathering: boolean;
  continuesFromPreviousDay: boolean;
  continuesToNextDay: boolean;
};

// Pixels taken off the event block's top and bottom, and off the top of the
// optional gathering band. The separator from an adjacent event belongs to the
// outermost piece: the commute band, else the optional gathering band, else
// the block itself. Continuation markers also take room inside the block.
export const getVerticalInsets = ({
  trimTop,
  trimBottom,
  hasCommuteBefore,
  hasCommuteAfter,
  hasOptionalGathering,
  continuesFromPreviousDay,
  continuesToNextDay,
}: VerticalInsetOptions) => {
  const outerBandTop =
    trimTop && !hasCommuteBefore ? EVENT_SEPARATOR_PIXELS : 0;
  return {
    outerBandTop,
    top:
      (hasOptionalGathering ? 0 : outerBandTop) +
      (continuesFromPreviousDay ? CONTINUATION_MARKER_HEIGHT : 0),
    bottom:
      (trimBottom && !hasCommuteAfter ? EVENT_SEPARATOR_PIXELS : 0) +
      (continuesToNextDay ? CONTINUATION_MARKER_HEIGHT : 0),
  };
};

export const describeApproximateEdges = (start: boolean, end: boolean) => {
  if (start && end) return "Approximate start and end";
  if (start) return "Approximate start";
  if (end) return "Approximate end";
  return undefined;
};
