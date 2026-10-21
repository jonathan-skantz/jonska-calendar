import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, cleanup, renderHook } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { api } from "../../lib/api";
import { queryKeys } from "../../lib/queryKeys";
import { icsExportUrl, useIcsImport } from "./useIcsImport";

vi.mock("../../lib/api", () => ({ api: vi.fn() }));

const mockedApi = vi.mocked(api);
const result = { imported: 1, skipped: [], warnings: [] };

const createHook = () => {
  const queryClient = new QueryClient();
  const invalidate = vi.spyOn(queryClient, "invalidateQueries");
  const wrapper = ({ children }: { children: React.ReactNode }) => (
    <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
  );
  const hook = renderHook(() => useIcsImport(), { wrapper });
  return { ...hook, queryClient, invalidate };
};

const timeZone = Intl.DateTimeFormat().resolvedOptions().timeZone;

describe("useIcsImport", () => {
  beforeEach(() => {
    mockedApi.mockReset();
    mockedApi.mockResolvedValue(result);
  });
  afterEach(cleanup);

  it("previews without refreshing the calendar", async () => {
    const { result: hook, queryClient, invalidate } = createHook();

    await act(async () => {
      await hook.current.mutateAsync({
        ics: "BEGIN:VCALENDAR",
        tagId: "",
        dryRun: true,
      });
    });

    expect(mockedApi).toHaveBeenCalledWith("/api/import/ics", {
      method: "POST",
      body: JSON.stringify({
        ics: "BEGIN:VCALENDAR",
        time_zone: timeZone,
        dry_run: true,
      }),
    });
    expect(invalidate).not.toHaveBeenCalled();
    queryClient.clear();
  });

  it("refreshes the calendar after a real import", async () => {
    const { result: hook, queryClient, invalidate } = createHook();

    await act(async () => {
      await hook.current.mutateAsync({
        ics: "BEGIN:VCALENDAR",
        tagId: "tag-2",
        dryRun: false,
      });
    });

    expect(mockedApi).toHaveBeenCalledWith("/api/import/ics", {
      method: "POST",
      body: JSON.stringify({
        ics: "BEGIN:VCALENDAR",
        time_zone: timeZone,
        tag_id: "tag-2",
        dry_run: false,
      }),
    });
    expect(invalidate).toHaveBeenCalledWith({ queryKey: queryKeys.workspace });
    queryClient.clear();
  });

  it("exports in the browser's time zone", () => {
    expect(icsExportUrl()).toBe(
      `/api/export.ics?time_zone=${encodeURIComponent(timeZone)}`,
    );
  });
});
