import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, cleanup, renderHook } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { api } from "../../lib/api";
import { queryKeys } from "../../lib/queryKeys";
import { useSubscriptionMutations } from "./useSubscriptions";

vi.mock("../../lib/api", () => ({ api: vi.fn() }));

const mockedApi = vi.mocked(api);

const createHook = () => {
  const queryClient = new QueryClient();
  const invalidate = vi.spyOn(queryClient, "invalidateQueries");
  const wrapper = ({ children }: { children: React.ReactNode }) => (
    <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
  );
  const hook = renderHook(() => useSubscriptionMutations(), { wrapper });
  return { ...hook, queryClient, invalidate };
};

describe("useSubscriptionMutations", () => {
  beforeEach(() => mockedApi.mockReset());
  afterEach(cleanup);

  it("reloads the subscriptions after a failed refresh", async () => {
    mockedApi.mockRejectedValueOnce(
      new Error("the calendar's server answered 410 Gone"),
    );
    const { result, queryClient, invalidate } = createHook();

    let error: string | null = null;
    await act(async () => {
      error = await result.current.refresh("sub-1");
    });

    expect(error).toBe("the calendar's server answered 410 Gone");
    expect(mockedApi).toHaveBeenCalledWith("/api/subscriptions/sub-1/refresh", {
      method: "POST",
      body: undefined,
    });
    expect(invalidate).toHaveBeenCalledWith({
      queryKey: queryKeys.subscriptions,
    });
    queryClient.clear();
  });

  it("refreshes the calendar after a successful refresh", async () => {
    mockedApi.mockResolvedValue({});
    const { result, queryClient, invalidate } = createHook();

    await act(async () => {
      expect(await result.current.refresh("sub-1")).toBeNull();
    });

    expect(invalidate).toHaveBeenCalledWith({ queryKey: queryKeys.workspace });
    queryClient.clear();
  });
});
