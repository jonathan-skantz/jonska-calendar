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

export const clearWorkspace = (queryClient: QueryClient) =>
  queryClient.removeQueries({ queryKey: queryKeys.workspace });
