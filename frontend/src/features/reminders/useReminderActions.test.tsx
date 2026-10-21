import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, renderHook, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { api } from "../../lib/api";
import type { Reminder } from "../../types";
import { useReminderActions } from "./useReminderActions";

vi.mock("../../lib/api", () => ({ api: vi.fn() }));

const mockedApi = vi.mocked(api);
const reminder = (id: string, eventId: string, offset: number): Reminder => ({
  id,
  event_id: eventId,
  offset_seconds: offset,
  remind_at: "2026-10-15T09:45:00Z",
  event_title: "Lecture",
  event_starts_at: "2026-10-15T10:00:00Z",
  occurrence_start: "2026-10-15T10:00:00Z",
  event_ends_at: "2026-10-15T11:00:00Z",
});

describe("useReminderActions", () => {
  beforeEach(() => {
    mockedApi.mockReset();
  });
  afterEach(cleanup);

  it("indexes the user's reminders by event", async () => {
    mockedApi.mockResolvedValue([
      reminder("r1", "event-1", 900),
      reminder("r2", "event-1", 3600),
      reminder("r3", "event-2", 0),
    ]);
    const queryClient = new QueryClient();
    const { result } = renderHook(() => useReminderActions(), {
      wrapper: ({ children }) => (
        <QueryClientProvider client={queryClient}>
          {children}
        </QueryClientProvider>
      ),
    });

    await waitFor(() => expect(result.current.reminders).toHaveLength(3));
    expect(result.current.remindersByEvent.get("event-2")).toHaveLength(1);
    expect(result.current.itemsFor("event-1")).toEqual([
      { id: "r1", offsetSeconds: 900 },
      { id: "r2", offsetSeconds: 3600 },
    ]);
    expect(result.current.itemsFor("unknown")).toEqual([]);
    queryClient.clear();
  });
});
