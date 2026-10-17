import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { api } from "../../lib/api";
import { useActionMutation } from "../../hooks/useActionMutation";
import {
  clearWorkspace,
  invalidateCurrentUserAndWorkspace,
  invalidateHousehold,
} from "../../lib/queryInvalidation";
import { queryKeys } from "../../lib/queryKeys";
import type { AuthMode, User } from "../../types";

export type Credentials = { mode: AuthMode; email: string; password: string };

const signIn = async ({ mode, email, password }: Credentials) => {
  const body = JSON.stringify({ email, password });
  await api<User>(`/api/auth/${mode === "login" ? "login" : "register"}`, {
    method: "POST",
    body,
  });
  // Registering does not start a session, so sign in right after.
  if (mode === "register") {
    await api("/api/auth/login", { method: "POST", body });
  }
};

export const useAuth = () => {
  const queryClient = useQueryClient();
  const currentUserQuery = useQuery({
    queryKey: queryKeys.currentUser,
    queryFn: () => api<User>("/api/auth/me"),
    retry: false,
  });
  const authenticate = useMutation({
    mutationFn: signIn,
    onSuccess: () => invalidateCurrentUserAndWorkspace(queryClient),
  });
  const logout = useMutation({
    mutationFn: () => api("/api/auth/logout", { method: "POST" }),
    // Set the user directly: a refetch would fail with 401, and a failed
    // refetch keeps the previous user.
    onSuccess: () => {
      queryClient.setQueryData<User | null>(queryKeys.currentUser, null);
      clearWorkspace(queryClient);
    },
  });
  const runProfileUpdate = useActionMutation(
    "Unable to update display name",
    async () => {
      await queryClient.invalidateQueries({ queryKey: queryKeys.currentUser });
      // Household members are listed by display name.
      await invalidateHousehold(queryClient);
    },
  );

  return {
    user: currentUserQuery.data ?? null,
    isLoading: currentUserQuery.isLoading,
    authenticate,
    logout,
    // Resolves to an error message, or null on success.
    updateDisplayName: (displayName: string) =>
      runProfileUpdate({
        path: "/api/auth/me",
        method: "PATCH",
        body: { display_name: displayName },
      }),
  };
};
