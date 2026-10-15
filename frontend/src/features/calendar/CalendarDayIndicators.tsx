import type { ElapsedOverlayState } from "../../lib/timeline";
import type { EventPreview } from "./timelineView";

type CalendarDayIndicatorsProps = {
  preview: EventPreview | null;
  elapsed: ElapsedOverlayState;
};

export const CalendarDayIndicators = ({
  preview,
  elapsed,
}: CalendarDayIndicatorsProps) => (
  <>
    {preview && (
      <div
        className="grid-entry event preview"
        style={{ top: preview.top, height: "4rem" }}
        aria-hidden="true"
      >
        <strong>New event</strong>
        <span>
          {preview.startsAt} – {preview.endsAt}
        </span>
      </div>
    )}
    {elapsed.kind === "past" && (
      <div className="day-elapsed-overlay past" aria-hidden="true" />
    )}
    {elapsed.kind === "today" && (
      <div
        className="day-elapsed-overlay today"
        style={{ height: elapsed.height }}
        aria-hidden="true"
      >
        <span className="current-time-label">{elapsed.label}</span>
      </div>
    )}
  </>
);
