import { RemindersDrawer } from "./RemindersDrawer";
import { useReminderActions } from "./useReminderActions";

type RemindersPanelProps = {
  now: Date;
  onOpenEvent: (eventId: string) => void;
  onClose: () => void;
};

export const RemindersPanel = ({
  now,
  onOpenEvent,
  onClose,
}: RemindersPanelProps) => {
  const { reminders, remove } = useReminderActions();
  return (
    <RemindersDrawer
      reminders={reminders}
      now={now}
      onOpenEvent={onOpenEvent}
      onRemove={(reminder) => remove(reminder.id)}
      onClose={onClose}
    />
  );
};
