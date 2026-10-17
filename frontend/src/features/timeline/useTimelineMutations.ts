import { useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "../../lib/api";
import { invalidateWorkspace } from "../../lib/queryInvalidation";
import {
  buildTimelineItemRequest,
  getEventTagChanges,
} from "./timelineRequests";
import type { Tag, TimelineEntry } from "../../types";

export type SaveItemInput = {
  type: "event" | "task";
  editingItem: TimelineEntry | null;
  form: FormData;
  mainTagId: string | null;
  selectedTagIds: string[];
  tags: Tag[];
};

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

const saveItem = async ({
  type,
  editingItem,
  form,
  mainTagId,
  selectedTagIds,
  tags,
}: SaveItemInput) => {
  if (type === "event" && editingItem?.type === "event") {
    await saveEventTags(editingItem.id, editingItem.tags ?? [], selectedTagIds);
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
    // The new event starts with only its main tag (or the default tag).
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
};

export const useTimelineMutations = () => {
  const queryClient = useQueryClient();
  const onSuccess = () => invalidateWorkspace(queryClient);

  return {
    saveItem: useMutation({ mutationFn: saveItem, onSuccess }),
    deleteItem: useMutation({
      mutationFn: (entry: TimelineEntry) =>
        api(`/api/timeline/${entry.id}`, { method: "DELETE" }),
      onSuccess,
    }),
    toggleTask: useMutation({
      mutationFn: (entry: TimelineEntry) =>
        api(`/api/tasks/${entry.id}`, {
          method: "PATCH",
          body: JSON.stringify({ is_completed: !entry.is_completed }),
        }),
      onSuccess,
    }),
  };
};
