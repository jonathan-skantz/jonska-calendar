import { useState } from "react";
import type { TimelineEntry } from "../../types";
import { CustomizationForm } from "../receivedCustomization/CustomizationForm";
import { useCustomizationMutations } from "../receivedCustomization/useCustomizationMutations";
import { useReminderActions } from "../reminders/useReminderActions";
import { SharedEventDrawer } from "./SharedEventDrawer";

type SharedEventPanelProps = {
  entry: TimelineEntry;
  onClose: () => void;
};

// An event shared with the user, or from a calendar they subscribe to, with
// their own version of it and, for shared events, their reminders.
export const SharedEventPanel = ({ entry, onClose }: SharedEventPanelProps) => {
  const reminders = useReminderActions();
  const customization = useCustomizationMutations(entry);
  const [editing, setEditing] = useState(false);
  const [error, setError] = useState<string | null>(null);
  // Reminders on subscribed events are not offered yet.
  const reminderProps = entry.subscription
    ? {}
    : {
        reminders: reminders.itemsFor(entry.id),
        reminderError: reminders.error,
        onAddReminder: (offset: number) => reminders.add(entry.id, offset),
        onRemoveReminder: (reminder: { id: string }) =>
          reminders.remove(reminder.id),
      };
  return (
    <SharedEventDrawer
      entry={entry}
      onClose={onClose}
      {...reminderProps}
      customizationError={editing ? null : error}
      onCustomize={() => setEditing(true)}
      onResetCustomization={async () => setError(await customization.reset())}
      editor={
        editing ? (
          <CustomizationForm
            entry={entry}
            onSave={customization.save}
            onCancel={() => setEditing(false)}
          />
        ) : undefined
      }
    />
  );
};
