import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, cleanup, renderHook } from "@testing-library/react";
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
  const onError = vi.fn();
  const onDeleted = vi.fn();
  const hook = renderHook(() => useTagMutations(onError, onDeleted), { wrapper });
  return { ...hook, queryClient, onError, onDeleted };
};

describe("useTagMutations", () => {
  beforeEach(() => mockedApi.mockReset());
  afterEach(cleanup);

  it("updates the user's default tag", async () => {
    mockedApi.mockResolvedValue(undefined);
    const { result, queryClient } = createHook();

    await act(async () => result.current.updateDefaultTag(tag.id));

    expect(mockedApi).toHaveBeenCalledWith("/api/tags/default", {
      method: "PUT",
      body: JSON.stringify({ tag_id: tag.id }),
    });
    queryClient.clear();
  });

  it("deletes a tag only after confirmation and clears its local visibility", async () => {
    mockedApi.mockResolvedValue(undefined);
    const {
      result,
      queryClient,
      onDeleted,
    } = createHook();
    const confirm = vi.fn().mockReturnValue(false);

    await act(async () => result.current.deleteTag(tag, confirm));
    expect(confirm).toHaveBeenCalledOnce();
    expect(mockedApi).not.toHaveBeenCalled();
    expect(onDeleted).not.toHaveBeenCalled();

    confirm.mockReturnValue(true);
    await act(async () => result.current.deleteTag(tag, confirm));

    expect(mockedApi).toHaveBeenCalledWith(`/api/tags/${tag.id}`, {
      method: "DELETE",
    });
    expect(onDeleted).toHaveBeenCalledWith(tag.id);
    queryClient.clear();
  });
});
