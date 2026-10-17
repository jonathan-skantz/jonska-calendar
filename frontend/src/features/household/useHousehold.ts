import { useQuery, useQueryClient } from "@tanstack/react-query";
import { api, sendForError } from "../../lib/api";
import { invalidateHousehold } from "../../lib/queryInvalidation";
import { queryKeys } from "../../lib/queryKeys";
import type { Household } from "../../types";

// The API answers null when the user has no household.
export const useHousehold = (enabled: boolean) =>
  useQuery({
    queryKey: queryKeys.household,
    queryFn: () => api<Household | null>("/api/household"),
    enabled,
  });

export type HouseholdDetails = {
  name: string;
  transit_card_count: number;
};

// Each action resolves to an error message, or null when it succeeded.
export const useHouseholdMutations = () => {
  const queryClient = useQueryClient();
  const send = (path: string, method: string, body?: unknown) =>
    sendForError(path, {
      method,
      body,
      fallbackMessage: "Unable to update household",
      onSuccess: () => invalidateHousehold(queryClient),
    });

  return {
    create: (details: HouseholdDetails) =>
      send("/api/household", "POST", details),
    update: (details: HouseholdDetails) =>
      send("/api/household", "PATCH", details),
    addMember: (email: string) =>
      send("/api/household/members", "POST", { email }),
    leave: () => send("/api/household/members/me", "DELETE"),
  };
};
