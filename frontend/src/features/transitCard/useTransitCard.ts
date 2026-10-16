import { useQuery, useQueryClient } from "@tanstack/react-query";
import { api } from "../../lib/api";
import { invalidateTransitCard } from "../../lib/queryInvalidation";
import { queryKeys } from "../../lib/queryKeys";
import { getRequestErrorMessage } from "../../lib/requestErrors";
import type { TransitCardDay } from "../../types";

export type TransitCardSubmission = {
  from: string;
  to: string;
  needed_dates: string[];
  maybe_dates: string[];
};

export const useTransitCardDays = (
  from: string,
  to: string,
  enabled: boolean,
) =>
  useQuery({
    queryKey: queryKeys.transitCardDays(from, to),
    queryFn: () =>
      api<TransitCardDay[]>(`/api/transit-card/days?from=${from}&to=${to}`),
    enabled,
  });

// Returns whether the save succeeded.
export const useSaveTransitCardDays = (onError: (message: string) => void) => {
  const queryClient = useQueryClient();

  return async (submission: TransitCardSubmission) => {
    try {
      await api("/api/transit-card/days", {
        method: "PUT",
        body: JSON.stringify(submission),
      });
      await invalidateTransitCard(queryClient);
      return true;
    } catch (requestError) {
      onError(
        getRequestErrorMessage(
          requestError,
          "Unable to save transit card days",
        ),
      );
      return false;
    }
  };
};
