import { afterEach, describe, expect, it, vi } from "vitest";
import { decideAction, sendAssistantMessage } from "./assistantApi";

describe("assistant API", () => {
  afterEach(() => vi.unstubAllGlobals());

  it("sends the browser's time zone and week start with the message", async () => {
    const fetch = vi.fn().mockResolvedValue(
      new Response(
        JSON.stringify({
          conversation_id: "c1",
          reply: "Hi",
          actions: [],
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

  it("posts a decision on an action", async () => {
    const fetch = vi
      .fn()
      .mockResolvedValue(new Response(JSON.stringify({}), { status: 200 }));
    vi.stubGlobal("fetch", fetch);

    await decideAction("c1", "a1", "apply");

    const [path, options] = fetch.mock.calls[0];
    expect(path).toBe("/api/assistant/conversations/c1/actions/a1");
    expect(JSON.parse(options.body)).toEqual({ decision: "apply" });
  });
});
