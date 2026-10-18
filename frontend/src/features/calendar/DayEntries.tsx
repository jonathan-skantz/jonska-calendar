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
}: DayEntriesProps) => {
  const { ghosts, busy } = layoutDayEntries(entries, date, now);
  const renderEntry = (item: DayEntryLayout) => (
    <CalendarEntry
      key={item.entry.id}
      {...item}
      reminders={remindersByEvent.get(item.entry.id)}
      onEdit={onEdit}
      onToggleTask={onToggleTask}
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
