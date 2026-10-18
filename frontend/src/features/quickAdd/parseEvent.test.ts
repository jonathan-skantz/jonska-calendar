import { beforeEach, describe, expect, it, vi } from "vitest";
import { api } from "../../lib/api";
import { readEventDraft } from "./parseEvent";

vi.mock("../../lib/api", () => ({ api: vi.fn() }));

const mockedApi = vi.mocked(api);

describe("readEventDraft", () => {
  beforeEach(() => {
    mockedApi.mockReset();
  });

  it("resolves the draft and collects every note on what wasn't placed", async () => {
    mockedApi.mockResolvedValue({
      draft: { title: "Dentist", start_time: "14:30" },
      warnings: ["Ignored the second time"],
      unresolved: ["next to the park"],
    });

    const draft = await readEventDraft("Dentist 14:30", 1, "2026-10-15");

    expect(mockedApi).toHaveBeenCalledWith("/api/events/parse", {
      method: "POST",
      body: JSON.stringify({ text: "Dentist 14:30" }),
    });
    expect(draft.values.title).toBe("Dentist");
    expect(draft.warnings.slice(0, 2)).toEqual([
      'Couldn\'t place "next to the park"',
      "Ignored the second time",
    ]);
  });
});
