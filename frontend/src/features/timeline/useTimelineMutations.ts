import { useQueryClient } from "@tanstack/react-query";
import { api } from "../../lib/api";
import { invalidateWorkspace } from "../../lib/queryInvalidation";
import { getRequestErrorMessage } from "../../lib/requestErrors";
import {
  buildTimelineItemRequest,
  getEventTagChanges,
} from "../../lib/timeline";
import type { Tag, TimelineEntry } from "../../types";

export const useTimelineMutations = (onError: (message: string) => void) => {
  const queryClient = useQueryClient();

  const saveEventTags = async (
    itemID: string,
    previous: Tag[],
    next: string[],
  ) => {
    const changes = getEventTagChanges(previous, next);
    await Promise.all([
      ...changes.add.map((tagID) =>
        api(`/api/timeline/${itemID}/tags`, {
          method: "POST",
          body: JSON.stringify({ tag_id: tagID }),
        }),
      ),
      ...changes.remove.map((tag) =>
        api(`/api/timeline/${itemID}/tags/${tag.id}`, { method: "DELETE" }),
      ),
    ]);
  };

  const toggleTask = async (entry: TimelineEntry) => {
    try {
      await api(`/api/tasks/${entry.id}`, {
        method: "PATCH",
        body: JSON.stringify({ is_completed: !entry.is_completed }),
      });
      await invalidateWorkspace(queryClient);
    } catch (requestError) {
      onError(getRequestErrorMessage(requestError, "Unable to update task"));
    }
  };

  const deleteItem = async (
    entry: TimelineEntry,
    confirm: () => boolean,
    onDeleted: () => void,
  ) => {
    if (!confirm()) return;
    try {
      await api(`/api/timeline/${entry.id}`, { method: "DELETE" });
      await invalidateWorkspace(queryClient);
      onDeleted();
    } catch (requestError) {
      onError(getRequestErrorMessage(requestError, "Unable to delete item"));
    }
  };

  const saveItem = async ({
    type,
    editingItem,
    form,
    mainTagId,
    selectedTagIds,
    tags,
    onSaved,
  }: {
    type: "event" | "task";
    editingItem: TimelineEntry | null;
    form: FormData;
    mainTagId: string | null;
    selectedTagIds: string[];
    tags: Tag[];
    onSaved: () => void;
  }) => {
    try {
      if (type === "event" && editingItem?.type === "event") {
        await saveEventTags(
          editingItem.id,
          editingItem.tags ?? [],
          selectedTagIds,
        );
      }
      const request = buildTimelineItemRequest(
        type,
        editingItem?.id ?? null,
        form,
        mainTagId,
      );
      const response = await api<{ id: string } | undefined>(request.path, {
        method: request.method,
        body: JSON.stringify(request.body),
      });
      if (type === "event" && editingItem?.type !== "event") {
        const initialTags = mainTagId
          ? tags.filter((tag) => tag.id === mainTagId)
          : tags.filter((tag) => tag.is_default);
        await saveEventTags(response?.id ?? "", initialTags, selectedTagIds);
        // Reminders added while composing a new event need its id first.
        await Promise.all(
          form.getAll("reminder_offset").map((offset) =>
            api(`/api/events/${response?.id}/reminders`, {
              method: "POST",
              body: JSON.stringify({ offset_seconds: Number(offset) }),
            }),
          ),
        );
      }
      await invalidateWorkspace(queryClient);
      onSaved();
    } catch (requestError) {
      onError(getRequestErrorMessage(requestError, "Unable to create item"));
    }
  };

  return { toggleTask, deleteItem, saveItem };
};
