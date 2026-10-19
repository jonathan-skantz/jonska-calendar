import { afterEach, describe, expect, it, vi } from "vitest";
import { proposalToDraft, sendAssistantMessage } from "./assistantApi";

describe("assistant API", () => {
  afterEach(() => vi.unstubAllGlobals());

  it("sends the browser's time zone and week start with the message", async () => {
    const fetch = vi.fn().mockResolvedValue(
      new Response(
        JSON.stringify({
          conversation_id: "c1",
          reply: "Hi",
          proposals: [],
          steps: [],
          ended: false,
        }),
        { status: 200 },
      ),
    );
    vi.stubGlobal("fetch", fetch);

    await sendAssistantMessage("hello", "c1", 1);

    const [path, options] = fetch.mock.calls[0];
    expect(path).toBe("/api/assistant/messages");
    expect(JSON.parse(options.body)).toEqual({
      conversation_id: "c1",
      text: "hello",
      time_zone: Intl.DateTimeFormat().resolvedOptions().timeZone,
      week_start: 1,
    });
  });

  it("turns a proposal into an event form draft", () => {
    expect(
      proposalToDraft({
        title: "Lunch",
        starts_at: "2026-10-26T12:00:00+01:00",
        ends_at: "2026-10-26T13:00:00+01:00",
        is_all_day: false,
        kind: "attendable",
        tag_ids: ["tag-1"],
        reminder_minutes: [30],
      }),
    ).toEqual({
      values: {
        title: "Lunch",
        starts_at: "2026-10-26T12:00:00+01:00",
        ends_at: "2026-10-26T13:00:00+01:00",
        is_all_day: false,
        location: undefined,
        kind: "attendable",
      },
      tagIds: ["tag-1"],
      reminderOffsets: [1800],
      warnings: [],
    });
  });
});
