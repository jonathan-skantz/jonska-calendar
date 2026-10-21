import { getErrorText } from "../../lib/requestErrors";
import type { Tag } from "../../types";
import { useTags } from "./useTags";
import { TagEditorDrawer } from "./TagEditorDrawer";
import { useTagMutations } from "./useTagMutations";

type TagEditorPanelProps = {
  onClose: () => void;
  // A deleted tag can no longer be hidden, so its filter state goes too.
  onTagDeleted: (tagId: string) => void;
};

// The tag editor with its own tag updates. Mounted only while open, so each
// opening starts without errors from the last one.
export const TagEditorPanel = ({
  onClose,
  onTagDeleted,
}: TagEditorPanelProps) => {
  const tags = useTags().data ?? [];
  const { updateTag, updateDefaultTag, deleteTag } =
    useTagMutations(onTagDeleted);
  const remove = (tag: Tag) => {
    if (window.confirm(`Delete tag “${tag.name}”?`)) deleteTag.mutate(tag);
  };

  return (
    <TagEditorDrawer
      tags={tags}
      error={getErrorText(
        updateTag.error ?? updateDefaultTag.error ?? deleteTag.error,
        "Unable to update tags",
      )}
      onClose={onClose}
      onUpdate={(tag, name, color) => {
        if (name) updateTag.mutate({ tag, name, color });
      }}
      onUpdateDefault={(tag) => updateDefaultTag.mutate(tag.id)}
      onDelete={remove}
    />
  );
};
