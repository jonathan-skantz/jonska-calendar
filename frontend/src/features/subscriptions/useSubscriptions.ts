import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useActionMutation } from "../../hooks/useActionMutation";
import { api } from "../../lib/api";
import { browserTimeZone } from "../../lib/calendar";
import { invalidateWorkspace } from "../../lib/queryInvalidation";
import { queryKeys } from "../../lib/queryKeys";
import type { Subscription } from "../../types";

export const useSubscriptions = () =>
  useQuery({
    queryKey: queryKeys.subscriptions,
    queryFn: () => api<Subscription[]>("/api/subscriptions"),
  });

export type SubscriptionDetails = {
  // Empty on create means the calendar's own name.
  name: string;
  color: string;
};

// Each action resolves to an error message, or null when it succeeded. They
// all change which events the calendar shows, so they refresh it.
export const useSubscriptionMutations = () => {
  const queryClient = useQueryClient();
  const run = useActionMutation("Unable to update the subscription", () =>
    invalidateWorkspace(queryClient),
  );
  const path = (id: string) => `/api/subscriptions/${encodeURIComponent(id)}`;

  return {
    // The server reads floating times and all-day dates in the browser's
    // zone, and fetches the calendar before saving it.
    add: (url: string, details: SubscriptionDetails) =>
      run({
        path: "/api/subscriptions",
        method: "POST",
        body: { url, ...details, time_zone: browserTimeZone() },
      }),
    update: (id: string, details: SubscriptionDetails) =>
      run({ path: path(id), method: "PATCH", body: details }),
    refresh: (id: string) =>
      run({ path: `${path(id)}/refresh`, method: "POST" }),
    remove: (id: string) => run({ path: path(id), method: "DELETE" }),
  };
};
