import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { api } from "../../lib/api";
import { invalidateTransitCard } from "../../lib/queryInvalidation";
import { queryKeys } from "../../lib/queryKeys";
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

export const useSaveTransitCardDays = () => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (submission: TransitCardSubmission) =>
      api("/api/transit-card/days", {
        method: "PUT",
        body: JSON.stringify(submission),
      }),
    onSuccess: () => invalidateTransitCard(queryClient),
  });
};
