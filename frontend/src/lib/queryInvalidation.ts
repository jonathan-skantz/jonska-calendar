import type { QueryClient } from "@tanstack/react-query";
import { queryKeys } from "./queryKeys";

export const invalidateReminders = (queryClient: QueryClient) =>
  queryClient.invalidateQueries({ queryKey: queryKeys.reminders });

// Reminders carry event fields (title, times) and disappear with their event
// or share, so any timeline change refreshes them too.
export const invalidateWorkspace = (queryClient: QueryClient) =>
  Promise.all([
    queryClient.invalidateQueries({ queryKey: queryKeys.workspace }),
    invalidateReminders(queryClient),
  ]);

export const invalidateCurrentUserAndWorkspace = async (
  queryClient: QueryClient,
) => {
  await queryClient.invalidateQueries({ queryKey: queryKeys.currentUser });
  await invalidateWorkspace(queryClient);
};

export const invalidateTransitCard = (queryClient: QueryClient) =>
  queryClient.invalidateQueries({ queryKey: queryKeys.transitCard });

// Membership and card count change which declarations are visible and
// whether a day is a conflict, so household changes refresh both.
export const invalidateHousehold = async (queryClient: QueryClient) => {
  await queryClient.invalidateQueries({ queryKey: queryKeys.household });
  await invalidateTransitCard(queryClient);
};

export const clearWorkspace = (queryClient: QueryClient) => {
  queryClient.removeQueries({ queryKey: queryKeys.workspace });
  queryClient.removeQueries({ queryKey: queryKeys.transitCard });
  queryClient.removeQueries({ queryKey: queryKeys.household });
  queryClient.removeQueries({ queryKey: queryKeys.reminders });
};
