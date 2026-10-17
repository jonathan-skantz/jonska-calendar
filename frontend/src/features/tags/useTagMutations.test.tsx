import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, cleanup, renderHook, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { api } from "../../lib/api";
import { useTagMutations } from "./useTagMutations";

vi.mock("../../lib/api", () => ({ api: vi.fn() }));

const mockedApi = vi.mocked(api);
const tag = {
  id: "tag-1",
  name: "Work",
  color: "#336699",
  is_default: false,
};

const createHook = () => {
  const queryClient = new QueryClient();
  const wrapper = ({ children }: { children: React.ReactNode }) => (
    <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
  );
  const onDeleted = vi.fn();
  const hook = renderHook(() => useTagMutations(onDeleted), { wrapper });
  return { ...hook, queryClient, onDeleted };
};

describe("useTagMutations", () => {
  beforeEach(() => {
    mockedApi.mockReset();
  });
  afterEach(cleanup);

  it("updates the user's default tag", async () => {
    mockedApi.mockResolvedValue(undefined);
    const { result, queryClient } = createHook();

    await act(async () => {
      await result.current.updateDefaultTag.mutateAsync(tag.id);
    });

    expect(mockedApi).toHaveBeenCalledWith("/api/tags/default", {
      method: "PUT",
      body: JSON.stringify({ tag_id: tag.id }),
    });
    queryClient.clear();
  });

  it("deletes a tag and clears its local visibility", async () => {
    mockedApi.mockResolvedValue(undefined);
    const { result, queryClient, onDeleted } = createHook();

    await act(async () => {
      await result.current.deleteTag.mutateAsync(tag);
    });

    expect(mockedApi).toHaveBeenCalledWith(`/api/tags/${tag.id}`, {
      method: "DELETE",
    });
    expect(onDeleted).toHaveBeenCalledWith(tag.id);
    queryClient.clear();
  });

  it("keeps a tag's local visibility when deleting fails", async () => {
    mockedApi.mockImplementation(async () => {
      throw new Error("Tag not found");
    });
    const { result, queryClient, onDeleted } = createHook();

    await act(async () => {
      await result.current.deleteTag.mutateAsync(tag).catch(() => {});
    });

    expect(onDeleted).not.toHaveBeenCalled();
    await waitFor(() =>
      expect(result.current.deleteTag.error?.message).toBe("Tag not found"),
    );
    queryClient.clear();
  });
});
