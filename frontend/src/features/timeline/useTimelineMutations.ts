import { useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "../../lib/api";
import { invalidateWorkspace } from "../../lib/queryInvalidation";
import { queryKeys } from "../../lib/queryKeys";
import { entryKey } from "../../lib/timeline";
import {
  buildMoveRequest,
  buildTimelineItemRequest,
  getEventTagChanges,
} from "./timelineRequests";
import type { Tag, TimelineEntry } from "../../types";

type TimelineSnapshot = [readonly unknown[], TimelineEntry[] | undefined][];

export type SaveItemInput = {
  type: "event" | "task";
  editingItem: TimelineEntry | null;
  form: FormData;
  mainTagId: string | null;
  selectedTagIds: string[];
  tags: Tag[];
};

type EventTagChanges = ReturnType<typeof getEventTagChanges>;

const addEventTags = (itemID: string, changes: EventTagChanges) =>
  Promise.all(
    changes.add.map((tagID) =>
      api(`/api/timeline/${itemID}/tags`, {
        method: "POST",
        body: JSON.stringify({ tag_id: tagID }),
      }),
    ),
  );

const removeEventTags = (itemID: string, changes: EventTagChanges) =>
  Promise.all(
    changes.remove.map((tag) =>
      api(`/api/timeline/${itemID}/tags/${tag.id}`, { method: "DELETE" }),
    ),
  );

const saveItem = async ({
  type,
  editingItem,
  form,
  mainTagId,
  selectedTagIds,
  tags,
}: SaveItemInput) => {
  // Add new tags before the save so the new main tag is attached when it is
  // saved. Remove old tags only after it, because the server refuses to
  // remove an event's current main tag.
  const editedEvent =
    type === "event" && editingItem?.type === "event" ? editingItem : null;
  const tagChanges = editedEvent
    ? getEventTagChanges(editedEvent.tags ?? [], selectedTagIds)
    : null;
  if (editedEvent && tagChanges) {
    await addEventTags(editedEvent.id, tagChanges);
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
  if (editedEvent && tagChanges) {
    await removeEventTags(editedEvent.id, tagChanges);
  }
  if (type === "event" && editingItem?.type !== "event") {
    // The new event starts with only its main tag (or the default tag).
    const initialTags = mainTagId
      ? tags.filter((tag) => tag.id === mainTagId)
      : tags.filter((tag) => tag.is_default);
    const newEventChanges = getEventTagChanges(initialTags, selectedTagIds);
    await addEventTags(response?.id ?? "", newEventChanges);
    await removeEventTags(response?.id ?? "", newEventChanges);
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
  const refreshWorkspace = () => invalidateWorkspace(queryClient);
  // Shows a change to one entry at once, in every loaded range. A failed
  // request puts the old entries back, and either way the calendar is
  // refetched afterwards.
  const optimisticEntryUpdate = (
    toUpdated: (input: TimelineEntry) => TimelineEntry,
  ) => ({
    onMutate: async (input: TimelineEntry) => {
      const updated = toUpdated(input);
      await queryClient.cancelQueries({ queryKey: queryKeys.timelines });
      const previous: TimelineSnapshot = queryClient.getQueriesData<
        TimelineEntry[]
      >({ queryKey: queryKeys.timelines });
      queryClient.setQueriesData<TimelineEntry[]>(
        { queryKey: queryKeys.timelines },
        (entries) =>
          entries?.map((item) =>
            entryKey(item) === entryKey(updated) ? updated : item,
          ),
      );
      return { previous };
    },
    onError: (
      _error: Error,
      _input: TimelineEntry,
      context?: { previous: TimelineSnapshot },
    ) => {
      for (const [queryKey, entries] of context?.previous ?? []) {
        queryClient.setQueryData(queryKey, entries);
      }
    },
    onSettled: refreshWorkspace,
  });

  return {
    saveItem: useMutation({
      mutationFn: saveItem,
      onSuccess: refreshWorkspace,
    }),
    deleteItem: useMutation({
      mutationFn: (entry: TimelineEntry) =>
        api(`/api/timeline/${entry.id}`, { method: "DELETE" }),
      onSuccess: refreshWorkspace,
    }),
    toggleTask: useMutation({
      mutationFn: (entry: TimelineEntry) =>
        api(`/api/tasks/${entry.id}`, {
          method: "PATCH",
          body: JSON.stringify({ is_completed: !entry.is_completed }),
        }),
      ...optimisticEntryUpdate((entry) => ({
        ...entry,
        is_completed: !entry.is_completed,
      })),
    }),
    // Takes the item at its new time, as dragged in the week view.
    moveItem: useMutation({
      mutationFn: (moved: TimelineEntry) => {
        const { path, body } = buildMoveRequest(moved);
        return api(path, { method: "PUT", body: JSON.stringify(body) });
      },
      ...optimisticEntryUpdate((moved) => moved),
    }),
  };
};
