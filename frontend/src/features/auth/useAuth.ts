import { useState } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { api } from "../../lib/api";
import {
  clearWorkspace,
  invalidateCurrentUserAndWorkspace,
} from "../../lib/queryInvalidation";
import { getRequestErrorMessage } from "../../lib/requestErrors";
import { queryKeys } from "../../lib/queryKeys";
import type { AuthMode, User } from "../../types";

export const useAuth = () => {
  const queryClient = useQueryClient();
  const [error, setError] = useState("");
  const currentUserQuery = useQuery({
    queryKey: queryKeys.currentUser,
    queryFn: () => api<User>("/api/auth/me"),
    retry: false,
  });

  const authenticate = async (
    mode: AuthMode,
    email: string,
    password: string,
  ) => {
    setError("");
    try {
      const path = `/api/auth/${mode === "login" ? "login" : "register"}`;
      await api<User>(path, {
        method: "POST",
        body: JSON.stringify({ email, password }),
      });
      if (mode === "register") {
        await api("/api/auth/login", {
          method: "POST",
          body: JSON.stringify({ email, password }),
        });
      }
      await invalidateCurrentUserAndWorkspace(queryClient);
    } catch (requestError) {
      setError(getRequestErrorMessage(requestError, "Unable to sign in"));
    }
  };

  const logout = async () => {
    await api("/api/auth/logout", { method: "POST" });
    await queryClient.invalidateQueries({ queryKey: queryKeys.currentUser });
    clearWorkspace(queryClient);
  };

  return {
    user: currentUserQuery.data ?? null,
    isLoading: currentUserQuery.isLoading,
    error,
    authenticate,
    logout,
  };
};
