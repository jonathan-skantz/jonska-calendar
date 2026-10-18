import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { api } from "../../lib/api";
import { dateKey } from "../../lib/calendar";
import { invalidateTransitCard } from "../../lib/queryInvalidation";
import { queryKeys } from "../../lib/queryKeys";
import type { TransitCardDay } from "../../types";
import { useHousehold } from "../household/useHousehold";
import { getTransitCardDayViews } from "./transitCardDays";

export type TransitCardSubmission = {
  from: string;
  to: string;
  needed_dates: string[];
  maybe_dates: string[];
};

export const useTransitCardDays = (from: string, to: string) =>
  useQuery({
    queryKey: queryKeys.transitCardDays(from, to),
    queryFn: () =>
      api<TransitCardDay[]>(`/api/transit-card/days?from=${from}&to=${to}`),
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

// The shown days' transit card answers for the user and their household, or
// null until the days load (every day would otherwise look unreported).
export const useTransitCardViews = (
  dates: Date[],
  today: Date,
  userId: string,
) => {
  const days = useTransitCardDays(
    dateKey(dates[0]),
    dateKey(dates[dates.length - 1]),
  ).data;
  const members = useHousehold().data?.members;
  return days
    ? getTransitCardDayViews(dates, days, today, userId, members)
    : null;
};
