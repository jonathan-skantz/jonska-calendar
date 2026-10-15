import { useQueryClient } from "@tanstack/react-query";
import { api } from "../../lib/api";
import { invalidateWorkspace } from "../../lib/queryInvalidation";
import { getRequestErrorMessage } from "../../lib/requestErrors";
import type { Tag } from "../../types";

export const useTagMutations = (
  onError: (message: string) => void,
  onDeleted: (tagID: string) => void,
) => {
  const queryClient = useQueryClient();

  const createTag = async (name: string) => {
    try {
      await api<Tag>("/api/tags", {
        method: "POST",
        body: JSON.stringify({ name }),
      });
      await invalidateWorkspace(queryClient);
    } catch (requestError) {
      onError(getRequestErrorMessage(requestError, "Unable to create tag"));
    }
  };

  const updateDefaultTag = async (tagID: string) => {
    try {
      await api("/api/tags/default", {
        method: "PUT",
        body: JSON.stringify({ tag_id: tagID }),
      });
      await invalidateWorkspace(queryClient);
    } catch (requestError) {
      onError(getRequestErrorMessage(requestError, "Unable to set default tag"));
    }
  };

  const updateTag = async (tag: Tag, name: string, color: string) => {
    if (!name) return;
    try {
      await api(`/api/tags/${tag.id}`, {
        method: "PATCH",
        body: JSON.stringify({ name, color }),
      });
      await invalidateWorkspace(queryClient);
    } catch (requestError) {
      onError(getRequestErrorMessage(requestError, "Unable to update tag"));
    }
  };

  const deleteTag = async (tag: Tag, confirm: () => boolean) => {
    if (!confirm()) return;
    try {
      await api(`/api/tags/${tag.id}`, { method: "DELETE" });
      onDeleted(tag.id);
      await invalidateWorkspace(queryClient);
    } catch (requestError) {
      onError(getRequestErrorMessage(requestError, "Unable to delete tag"));
    }
  };

  return { createTag, updateDefaultTag, updateTag, deleteTag };
};
