import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, cleanup, renderHook, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { api } from "../../lib/api";
import { useAuth } from "./useAuth";

vi.mock("../../lib/api", () => ({ api: vi.fn() }));

const mockedApi = vi.mocked(api);
const user = { id: "user-1", email: "person@example.com" };

describe("useAuth", () => {
  beforeEach(() => {
    mockedApi.mockReset();
  });

  afterEach(cleanup);

  it("logs in after successful registration", async () => {
    mockedApi.mockImplementation(async (path) => {
      if (path === "/api/auth/me") return user;
      return undefined;
    });
    const queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false } },
    });
    const wrapper = ({ children }: { children: React.ReactNode }) => (
      <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
    );
    const { result } = renderHook(() => useAuth(), { wrapper });
    await waitFor(() => expect(result.current.user).toEqual(user));

    await act(async () => {
      await result.current.authenticate("register", user.email, "secret");
    });

    const authRequests = mockedApi.mock.calls
      .filter(([path]) => path !== "/api/auth/me")
      .map(([path]) => path);
    expect(authRequests).toEqual(["/api/auth/register", "/api/auth/login"]);
    expect(mockedApi).toHaveBeenCalledWith("/api/auth/register", {
      method: "POST",
      body: JSON.stringify({ email: user.email, password: "secret" }),
    });
    queryClient.clear();
  });

  it("returns login failures for the authentication screen", async () => {
    mockedApi.mockImplementation(async (path) => {
      if (path === "/api/auth/me") throw new Error("Not signed in");
      throw new Error("Invalid email or password");
    });
    const queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false } },
    });
    const wrapper = ({ children }: { children: React.ReactNode }) => (
      <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
    );
    const { result } = renderHook(() => useAuth(), { wrapper });
    await waitFor(() => expect(result.current.isLoading).toBe(false));

    await act(async () => {
      await result.current.authenticate("login", user.email, "wrong");
    });

    expect(result.current.error).toBe("Invalid email or password");
    queryClient.clear();
  });
});
