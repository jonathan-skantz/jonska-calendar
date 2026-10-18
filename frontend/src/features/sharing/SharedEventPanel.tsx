import type { TimelineEntry } from "../../types";
import { useReminderActions } from "../reminders/useReminderActions";
import { SharedEventDrawer } from "./SharedEventDrawer";

type SharedEventPanelProps = {
  entry: TimelineEntry;
  onClose: () => void;
};

// An event shared with the user, with the reminders they keep for it.
export const SharedEventPanel = ({ entry, onClose }: SharedEventPanelProps) => {
  const reminders = useReminderActions();
  return (
    <SharedEventDrawer
      entry={entry}
      onClose={onClose}
      reminders={reminders.itemsFor(entry.id)}
      reminderError={reminders.error}
      onAddReminder={(offset) => reminders.add(entry.id, offset)}
      onRemoveReminder={(reminder) => reminders.remove(reminder.id)}
    />
  );
};
