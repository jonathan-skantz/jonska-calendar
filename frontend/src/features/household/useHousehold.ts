import { useQuery, useQueryClient } from "@tanstack/react-query";
import { api } from "../../lib/api";
import { invalidateHousehold } from "../../lib/queryInvalidation";
import { queryKeys } from "../../lib/queryKeys";
import { getRequestErrorMessage } from "../../lib/requestErrors";
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
  const send = async (path: string, method: string, body?: unknown) => {
    try {
      await api(path, {
        method,
        body: body === undefined ? undefined : JSON.stringify(body),
      });
      await invalidateHousehold(queryClient);
      return null;
    } catch (requestError) {
      return getRequestErrorMessage(requestError, "Unable to update household");
    }
  };

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
