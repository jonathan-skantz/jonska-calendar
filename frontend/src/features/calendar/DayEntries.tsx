import type { PointerEvent } from "react";
import { entryKey } from "../../lib/timeline";
import type { Reminder, TimelineEntry } from "../../types";
import { CalendarEntry } from "./CalendarEntry";
import { layoutDayEntries } from "./timelineView";
import type { DayEntryLayout } from "./timelineView";

type DayEntriesProps = {
  // The visible entries of the shown range; this picks the ones on `date`.
  entries: TimelineEntry[];
  date: Date;
  now: Date;
  remindersByEvent: Map<string, Reminder[]>;
  onEdit: (entry: TimelineEntry) => void;
  onToggleTask: (entry: TimelineEntry) => void;
  // Moving items by dragging; draggingKey is the entryKey of the item being
  // dragged.
  onDragStart?: (entry: TimelineEntry, event: PointerEvent) => void;
  draggingKey?: string;
};

// The timed entries of one day column. Ghosts render first so busy blocks
// paint over them.
export const DayEntries = ({
  entries,
  date,
  now,
  remindersByEvent,
  onEdit,
  onToggleTask,
  onDragStart,
  draggingKey,
}: DayEntriesProps) => {
  const { ghosts, busy } = layoutDayEntries(entries, date, now);
  const renderEntry = (item: DayEntryLayout) => (
    <CalendarEntry
      key={entryKey(item.entry)}
      {...item}
      reminders={remindersByEvent.get(item.entry.id)}
      onEdit={onEdit}
      onToggleTask={onToggleTask}
      onDragStart={onDragStart}
      isDragging={entryKey(item.entry) === draggingKey}
    />
  );
  // Elapsed entries are dimmed together in one wrapper per layer, so a
  // dimmed busy block still paints over every ghost.
  const renderLayer = (layer: DayEntryLayout[]) => {
    const elapsed = layer.filter((item) => item.fullyElapsed);
    return (
      <>
        {elapsed.length > 0 && (
          <div className="elapsed-event-group">{elapsed.map(renderEntry)}</div>
        )}
        {layer.filter((item) => !item.fullyElapsed).map(renderEntry)}
      </>
    );
  };

  return (
    <>
      {renderLayer(ghosts)}
      {renderLayer(busy)}
    </>
  );
};
