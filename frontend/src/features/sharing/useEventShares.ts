import { useQuery, useQueryClient } from "@tanstack/react-query";
import { api } from "../../lib/api";
import { useActionMutation } from "../../hooks/useActionMutation";
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
  const run = useActionMutation("Unable to update sharing", async () => {
    await queryClient.invalidateQueries({
      queryKey: queryKeys.eventShares(eventId ?? ""),
    });
    // share_count on the owner's timeline entry changes too.
    await invalidateWorkspace(queryClient);
  });
  const send = async (path: string, method: string, body?: unknown) =>
    eventId
      ? run({ path: `/api/events/${eventId}/shares${path}`, method, body })
      : "No event selected";

  return {
    add: (email: string) => send("", "POST", { recipient_email: email }),
    update: (shareId: string, fields: ShareFieldsUpdate) =>
      send(`/${shareId}`, "PUT", fields),
    remove: (shareId: string) => send(`/${shareId}`, "DELETE"),
  };
};
