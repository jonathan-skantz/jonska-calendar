import { useState } from "react";
import { getErrorText } from "../../lib/requestErrors";
import type { TimelineEntry } from "../../types";
import { useReminderActions } from "../reminders/useReminderActions";
import { ShareEventDrawer } from "../sharing/ShareEventDrawer";
import {
  useEventShareMutations,
  useEventShares,
} from "../sharing/useEventShares";
import { useTags } from "../tags/useTags";
import { useTimelineMutations } from "../timeline/useTimelineMutations";
import { ComposerDrawer } from "./ComposerDrawer";
import type { Composer } from "./useComposer";

type ComposerPanelProps = {
  composer: Composer;
  // The loaded entries, for the edited event's latest share count.
  entries: TimelineEntry[];
  selectedDate: string;
};

// The open composer with everything it changes on the server: saving and
// deleting the item, the saved event's reminders, and its sharing. Remount it
// per item (by key) so errors from a previous item don't carry over.
export const ComposerPanel = ({
  composer,
  entries,
  selectedDate,
}: ComposerPanelProps) => {
  const { editingItem } = composer;
  const tags = useTags().data ?? [];
  const { saveItem, deleteItem } = useTimelineMutations();
  const reminders = useReminderActions();
  const [sharingEvent, setSharingEvent] = useState<TimelineEntry | null>(null);
  const shares = useEventShares(sharingEvent?.id ?? null);
  const shareMutations = useEventShareMutations(sharingEvent?.id ?? null);

  // The share drawer is stacked on the composer, so it closes with it.
  const close = () => {
    setSharingEvent(null);
    composer.close();
  };
  const save = (form: FormData) =>
    saveItem.mutate(
      {
        type: composer.type,
        editingItem,
        form,
        mainTagId: composer.mainTagId,
        selectedTagIds: composer.selectedTagIds,
        tags,
      },
      { onSuccess: close },
    );
  const remove = (entry: TimelineEntry) => {
    if (window.confirm(`Delete “${entry.title}”?`)) {
      deleteItem.mutate(entry, { onSuccess: close });
    }
  };

  return (
    <>
      <ComposerDrawer
        type={composer.type}
        editingItem={editingItem}
        selectedDate={selectedDate}
        draftEventStart={composer.draftEventStart}
        tagSelection={{
          tags,
          selectedTagIds: composer.selectedTagIds,
          mainTagId: composer.mainTagId,
          onSelectedTagIdsChange: composer.setSelectedTagIds,
          onMainTagIdChange: composer.setMainTagId,
        }}
        error={getErrorText(
          saveItem.error ?? deleteItem.error,
          "Unable to save item",
        )}
        pending={saveItem.isPending}
        onSave={save}
        onClose={close}
        onTypeChange={composer.setType}
        onShare={
          editingItem?.type === "event"
            ? () => setSharingEvent(editingItem)
            : undefined
        }
        shareCount={
          entries.find((entry) => entry.id === editingItem?.id)?.share_count
        }
        savedReminders={
          editingItem?.type === "event"
            ? {
                items: reminders.itemsFor(editingItem.id),
                error: reminders.error,
                onAdd: (offset) => reminders.add(editingItem.id, offset),
                onRemove: (reminder) => reminders.remove(reminder.id),
              }
            : undefined
        }
        draft={composer.draft}
      />
      {editingItem && (
        <button
          className="danger-button drawer-delete-action"
          type="button"
          onClick={() => remove(editingItem)}
        >
          Delete item
        </button>
      )}
      {sharingEvent && (
        <ShareEventDrawer
          event={sharingEvent}
          shares={shares.data}
          onAdd={shareMutations.add}
          onUpdate={shareMutations.update}
          onRemove={async (share) =>
            window.confirm(
              `Stop sharing “${sharingEvent.title}” with ${share.recipient_display_name}?`,
            )
              ? shareMutations.remove(share.id)
              : null
          }
          onClose={() => setSharingEvent(null)}
        />
      )}
    </>
  );
};
