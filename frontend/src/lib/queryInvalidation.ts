import type { QueryClient } from "@tanstack/react-query";
import { queryKeys } from "./queryKeys";

export const invalidateWorkspace = (queryClient: QueryClient) =>
  queryClient.invalidateQueries({ queryKey: queryKeys.workspace });

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
};
