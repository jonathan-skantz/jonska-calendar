import { useState } from "react";
import { getErrorText } from "../../lib/requestErrors";
import type { TimelineEntry } from "../../types";
import { useReminderActions } from "../reminders/useReminderActions";
import { useCurrentUser } from "../auth/useAuth";
import { ShareEventDrawer } from "../sharing/ShareEventDrawer";
import {
  useEventShareMutations,
  useEventShares,
} from "../sharing/useEventShares";
import { useTags } from "../tags/useTags";
import { useRecurrenceScope } from "../calendar/useRecurrenceScope";
import { changesOnlyOccurrence } from "../timeline/timelineRequests";
import type { OccurrenceScope } from "../timeline/timelineRequests";
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
  // A demo account cannot share.
  const isDemo = useCurrentUser().user?.demo_expires_at !== undefined;
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
  const scopePrompt = useRecurrenceScope();
  // Saving or deleting an occurrence of a repeating event asks which
  // occurrences it applies to; cancelling keeps the form open.
  const save = async (form: FormData) => {
    const scope = editingItem?.occurrence_start
      ? await scopePrompt.ask({
          action: "save",
          allowThis: changesOnlyOccurrence(
            editingItem,
            form,
            composer.selectedTagIds,
            composer.mainTagId,
          ),
        })
      : undefined;
    if (scope === null) return;
    saveItem.mutate(
      {
        type: composer.type,
        editingItem,
        form,
        mainTagId: composer.mainTagId,
        selectedTagIds: composer.selectedTagIds,
        tags,
        scope,
      },
      { onSuccess: close },
    );
  };
  const remove = async (entry: TimelineEntry) => {
    let scope: OccurrenceScope | undefined;
    if (entry.occurrence_start) {
      const answer = await scopePrompt.ask({
        action: "delete",
        allowThis: true,
      });
      if (!answer) return;
      scope = answer;
    } else if (!window.confirm(`Delete “${entry.title}”?`)) {
      return;
    }
    deleteItem.mutate({ entry, scope }, { onSuccess: close });
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
          editingItem?.type === "event" && !isDemo
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
      {scopePrompt.dialog}
    </>
  );
};
