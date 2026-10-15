import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, cleanup, renderHook } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { api } from "../../lib/api";
import { useTimelineMutations } from "./useTimelineMutations";

vi.mock("../../lib/api", () => ({ api: vi.fn() }));

const mockedApi = vi.mocked(api);
const tags = [
  { id: "work", name: "Work", color: "#336699", is_default: true },
  { id: "home", name: "Home", color: "#993366", is_default: false },
];

describe("useTimelineMutations", () => {
  beforeEach(() => mockedApi.mockReset());
  afterEach(cleanup);

  it("creates an event with its main tag and attaches selected additional tags", async () => {
    mockedApi.mockImplementation(async (path) =>
      path === "/api/events" ? { id: "event-1" } : undefined,
    );
    const queryClient = new QueryClient();
    const wrapper = ({ children }: { children: React.ReactNode }) => (
      <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
    );
    const onError = vi.fn();
    const onSaved = vi.fn();
    const { result } = renderHook(() => useTimelineMutations(onError), { wrapper });
    const form = new FormData();
    form.set("title", "Planning");
    form.set("kind", "informational");
    form.set("starts_at", "2026-10-15T10:00");
    form.set("ends_at", "2026-10-15T11:00");

    await act(async () => {
      await result.current.saveItem({
        type: "event",
        editingItem: null,
        form,
        mainTagId: "work",
        selectedTagIds: ["work", "home"],
        tags,
        onSaved,
      });
    });

    expect(mockedApi).toHaveBeenNthCalledWith(1, "/api/events", {
      method: "POST",
      body: expect.stringContaining('"main_tag_id":"work"'),
    });
    expect(mockedApi).toHaveBeenNthCalledWith(2, "/api/timeline/event-1/tags", {
      method: "POST",
      body: JSON.stringify({ tag_id: "home" }),
    });
    expect(onSaved).toHaveBeenCalledOnce();
    expect(onError).not.toHaveBeenCalled();
    queryClient.clear();
  });
});
