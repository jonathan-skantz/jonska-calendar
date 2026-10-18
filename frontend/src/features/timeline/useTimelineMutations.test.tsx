import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, cleanup, renderHook, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { api } from "../../lib/api";
import { queryKeys } from "../../lib/queryKeys";
import type { TimelineEntry } from "../../types";
import { useTimelineMutations } from "./useTimelineMutations";

vi.mock("../../lib/api", () => ({ api: vi.fn() }));

const mockedApi = vi.mocked(api);
const tags = [
  { id: "work", name: "Work", color: "#336699", is_default: true },
  { id: "home", name: "Home", color: "#993366", is_default: false },
];

describe("useTimelineMutations", () => {
  beforeEach(() => {
    mockedApi.mockReset();
  });
  afterEach(cleanup);

  it("creates an event with its main tag and attaches selected additional tags", async () => {
    mockedApi.mockImplementation(async (path) =>
      path === "/api/events" ? { id: "event-1" } : undefined,
    );
    const queryClient = new QueryClient();
    const wrapper = ({ children }: { children: React.ReactNode }) => (
      <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
    );
    const { result } = renderHook(() => useTimelineMutations(), {
      wrapper,
    });
    const form = new FormData();
    form.set("title", "Planning");
    form.set("kind", "informational");
    form.set("starts_at", "2026-10-15T10:00");
    form.set("ends_at", "2026-10-15T11:00");

    await act(async () => {
      await result.current.saveItem.mutateAsync({
        type: "event",
        editingItem: null,
        form,
        mainTagId: "work",
        selectedTagIds: ["work", "home"],
        tags,
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
    await waitFor(() => expect(result.current.saveItem.isSuccess).toBe(true));
    queryClient.clear();
  });

  it("creates reminders queued on a new event after saving it", async () => {
    mockedApi.mockImplementation(async (path) =>
      path === "/api/events" ? { id: "event-1" } : undefined,
    );
    const queryClient = new QueryClient();
    const wrapper = ({ children }: { children: React.ReactNode }) => (
      <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
    );
    const { result } = renderHook(() => useTimelineMutations(), {
      wrapper,
    });
    const form = new FormData();
    form.set("title", "Lecture");
    form.set("kind", "attendable");
    form.set("starts_at", "2026-10-15T10:00");
    form.set("ends_at", "2026-10-15T12:00");
    form.append("reminder_offset", "900");
    form.append("reminder_offset", "86400");

    await act(async () => {
      await result.current.saveItem.mutateAsync({
        type: "event",
        editingItem: null,
        form,
        mainTagId: "work",
        selectedTagIds: ["work"],
        tags,
      });
    });

    expect(mockedApi).toHaveBeenCalledWith("/api/events/event-1/reminders", {
      method: "POST",
      body: JSON.stringify({ offset_seconds: 900 }),
    });
    expect(mockedApi).toHaveBeenCalledWith("/api/events/event-1/reminders", {
      method: "POST",
      body: JSON.stringify({ offset_seconds: 86400 }),
    });
    queryClient.clear();
  });

  it("moves the main tag before removing the old one when editing an event", async () => {
    mockedApi.mockImplementation(async () => undefined);
    const queryClient = new QueryClient();
    const wrapper = ({ children }: { children: React.ReactNode }) => (
      <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
    );
    const { result } = renderHook(() => useTimelineMutations(), { wrapper });
    const editingItem: TimelineEntry = {
      id: "event-1",
      type: "event",
      title: "Dinner",
      kind: "informational",
      tags: [tags[0]],
      is_completed: false,
    };
    const form = new FormData();
    form.set("title", "Dinner");
    form.set("kind", "informational");
    form.set("starts_at", "2026-10-15T18:00");
    form.set("ends_at", "2026-10-15T19:00");

    await act(async () => {
      await result.current.saveItem.mutateAsync({
        type: "event",
        editingItem,
        form,
        mainTagId: "home",
        selectedTagIds: ["home"],
        tags,
      });
    });

    expect(
      mockedApi.mock.calls.map(([path, options]) => [path, options?.method]),
    ).toEqual([
      ["/api/timeline/event-1/tags", "POST"],
      ["/api/events/event-1", "PUT"],
      ["/api/timeline/event-1/tags/work", "DELETE"],
    ]);
    queryClient.clear();
  });

  it("completes a task at once and rolls back when the request fails", async () => {
    const task: TimelineEntry = {
      id: "task-1",
      type: "task",
      title: "Follow up",
      deadline_at: "2026-10-16T15:00:00Z",
      is_completed: false,
    };
    let rejectToggle: (error: Error) => void = () => {};
    mockedApi.mockImplementation((path, options) => {
      if (path === "/api/tasks/task-1" && options?.method === "PATCH") {
        return new Promise((_, reject) => {
          rejectToggle = reject;
        });
      }
      // The refetch after the failure returns the server's unchanged data.
      if (path === "/api/timeline") return Promise.resolve([task]);
      return Promise.resolve([]);
    });
    const queryClient = new QueryClient();
    queryClient.setQueryData(queryKeys.workspace, { entries: [task], tags });
    const wrapper = ({ children }: { children: React.ReactNode }) => (
      <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
    );
    const { result } = renderHook(() => useTimelineMutations(), { wrapper });
    const completed = () =>
      queryClient.getQueryData<{ entries: TimelineEntry[] }>(
        queryKeys.workspace,
      )?.entries[0].is_completed;

    act(() => result.current.toggleTask.mutate(task));
    await waitFor(() => expect(completed()).toBe(true));

    await act(async () => rejectToggle(new Error("Task update failed")));
    await waitFor(() =>
      expect(result.current.toggleTask.error?.message).toBe(
        "Task update failed",
      ),
    );
    expect(completed()).toBe(false);
    queryClient.clear();
  });
});
