import { useQuery, useQueryClient } from "@tanstack/react-query";
import { api, sendForError } from "../../lib/api";
import { invalidateWorkspace } from "../../lib/queryInvalidation";
import { queryKeys } from "../../lib/queryKeys";
import type { EventShare } from "../../types";

// null clears a field so the recipient follows the event's own value again.
export type ShareFieldsUpdate = {
  title: string | null;
  description: string | null;
  url: string | null;
  location: string | null;
};

export const useEventShares = (eventId: string | null) =>
  useQuery({
    queryKey: queryKeys.eventShares(eventId ?? ""),
    queryFn: () => api<EventShare[]>(`/api/events/${eventId}/shares`),
    enabled: Boolean(eventId),
  });

// Each action resolves to an error message, or null when it succeeded.
export const useEventShareMutations = (eventId: string | null) => {
  const queryClient = useQueryClient();
  const send = async (path: string, method: string, body?: unknown) => {
    if (!eventId) return "No event selected";
    return sendForError(`/api/events/${eventId}/shares${path}`, {
      method,
      body,
      fallbackMessage: "Unable to update sharing",
      onSuccess: async () => {
        await queryClient.invalidateQueries({
          queryKey: queryKeys.eventShares(eventId),
        });
        // share_count on the owner's timeline entry changes too.
        await invalidateWorkspace(queryClient);
      },
    });
  };

  return {
    add: (email: string) => send("", "POST", { recipient_email: email }),
    update: (shareId: string, fields: ShareFieldsUpdate) =>
      send(`/${shareId}`, "PUT", fields),
    remove: (shareId: string) => send(`/${shareId}`, "DELETE"),
  };
};
