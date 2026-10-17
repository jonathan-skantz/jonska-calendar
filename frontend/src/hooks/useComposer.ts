import { useState } from "react";
import { dateKey } from "../lib/calendar";
import { getDefaultEventTagSelection } from "../lib/timeline";
import type { Tag, TimelineEntry } from "../types";
import type { ResolvedDraft } from "../features/quickAdd/resolveDraft";

type UseComposerOptions = {
  tags: Tag[];
  selectedDate: string;
  onSelectedDateChange: (date: string) => void;
  onResetErrors: () => void;
};

export const useComposer = ({
  tags,
  selectedDate,
  onSelectedDateChange,
  onResetErrors,
}: UseComposerOptions) => {
  const [showComposer, setShowComposer] = useState(false);
  const [type, setType] = useState<"event" | "task">("event");
  const [editingItem, setEditingItem] = useState<TimelineEntry | null>(null);
  const [selectedTagIds, setSelectedTagIds] = useState<string[]>([]);
  const [mainTagId, setMainTagId] = useState<string | null>(null);
  const [draftEventStart, setDraftEventStart] = useState<string | null>(null);
  const [draft, setDraft] = useState<ResolvedDraft | null>(null);

  const openNew = (startAt?: Date, nextDraft?: ResolvedDraft) => {
    setEditingItem(null);
    setType("event");
    setDraft(nextDraft ?? null);
    // Tags named in a draft replace the default selection; the first is main.
    const draftTagIds = (nextDraft?.tagIds ?? []).filter((id) =>
      tags.some((tag) => tag.id === id),
    );
    const selection = draftTagIds.length
      ? { tagIds: draftTagIds, mainTagId: draftTagIds[0] }
      : getDefaultEventTagSelection(tags);
    setSelectedTagIds(selection.tagIds);
    setMainTagId(selection.mainTagId);
    const draftStart = nextDraft?.values.starts_at;
    const start =
      startAt ??
      (draftStart ? new Date(draftStart) : new Date(`${selectedDate}T09:00`));
    onSelectedDateChange(dateKey(start));
    setDraftEventStart(start.toISOString());
    onResetErrors();
    setShowComposer(true);
  };

  const edit = (entry: TimelineEntry) => {
    setEditingItem(entry);
    setType(entry.type);
    setDraft(null);
    setDraftEventStart(null);
    onResetErrors();
    setSelectedTagIds(entry.tags?.map((tag) => tag.id) ?? []);
    setMainTagId(entry.main_tag_id ?? entry.tags?.[0]?.id ?? null);
    setShowComposer(true);
  };

  const close = () => {
    setShowComposer(false);
    setEditingItem(null);
  };

  return {
    showComposer,
    type,
    editingItem,
    selectedTagIds,
    mainTagId,
    draftEventStart,
    draft,
    setType,
    setEditingItem,
    setDraftEventStart,
    setSelectedTagIds,
    setMainTagId,
    openNew,
    edit,
    close,
    setShowComposer,
  };
};
