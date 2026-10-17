import { useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "../../lib/api";
import { invalidateWorkspace } from "../../lib/queryInvalidation";
import type { Tag } from "../../types";

export type TagUpdate = { tag: Tag; name: string; color: string };

export const useTagMutations = (onDeleted: (tagID: string) => void) => {
  const queryClient = useQueryClient();
  const onSuccess = () => invalidateWorkspace(queryClient);

  return {
    createTag: useMutation({
      mutationFn: (name: string) =>
        api<Tag>("/api/tags", {
          method: "POST",
          body: JSON.stringify({ name }),
        }),
      onSuccess,
    }),
    updateDefaultTag: useMutation({
      mutationFn: (tagID: string) =>
        api("/api/tags/default", {
          method: "PUT",
          body: JSON.stringify({ tag_id: tagID }),
        }),
      onSuccess,
    }),
    updateTag: useMutation({
      mutationFn: ({ tag, name, color }: TagUpdate) =>
        api(`/api/tags/${tag.id}`, {
          method: "PATCH",
          body: JSON.stringify({ name, color }),
        }),
      onSuccess,
    }),
    deleteTag: useMutation({
      mutationFn: (tag: Tag) =>
        api(`/api/tags/${tag.id}`, { method: "DELETE" }),
      onSuccess: (_, tag) => {
        // A deleted tag can no longer be hidden.
        onDeleted(tag.id);
        return invalidateWorkspace(queryClient);
      },
    }),
  };
};
