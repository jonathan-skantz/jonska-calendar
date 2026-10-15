import { useState } from "react";
import { dateKey } from "../lib/calendar";
import type { Tag, TimelineEntry } from "../types";

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

  const openNew = (startAt?: Date) => {
    setEditingItem(null);
    setType("event");
    setSelectedTagIds(tags.filter((tag) => tag.is_default).map((tag) => tag.id));
    setMainTagId(
      tags.find((tag) => tag.is_default)?.id ?? tags[0]?.id ?? null,
    );
    const start = startAt ?? new Date(`${selectedDate}T09:00`);
    onSelectedDateChange(dateKey(start));
    setDraftEventStart(start.toISOString());
    onResetErrors();
    setShowComposer(true);
  };

  const edit = (entry: TimelineEntry) => {
    setEditingItem(entry);
    setType(entry.type);
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
