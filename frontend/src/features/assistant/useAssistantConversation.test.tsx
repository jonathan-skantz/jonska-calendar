import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, cleanup, renderHook, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { api } from "../../lib/api";
import type { AssistantReply } from "../../types";
import { useAssistantConversation } from "./useAssistantConversation";

vi.mock("../../lib/api", () => ({ api: vi.fn() }));

const mockedApi = vi.mocked(api);
const reply = (text: string): AssistantReply => ({
  conversation_id: "c1",
  reply: text,
  actions: [],
  steps: [],
  ended: false,
});

const renderConversation = () => {
  const queryClient = new QueryClient({
    defaultOptions: { mutations: { retry: false } },
  });
  return renderHook(() => useAssistantConversation(1), {
    wrapper: ({ children }) => (
      <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
    ),
  });
};

const sentBody = (call: number) =>
  JSON.parse(mockedApi.mock.calls[call][1]!.body as string);

describe("useAssistantConversation", () => {
  beforeEach(() => mockedApi.mockReset());
  afterEach(cleanup);

  it("adds each message with its answer and continues the conversation", async () => {
    let answer: (value: AssistantReply) => void = () => {};
    mockedApi.mockReturnValueOnce(new Promise((resolve) => (answer = resolve)));
    const { result } = renderConversation();

    act(() => void result.current.send("lunch on Friday?"));
    await waitFor(() =>
      expect(result.current.pendingText).toBe("lunch on Friday?"),
    );
    expect(result.current.turns).toEqual([]);

    await act(async () => answer(reply("Friday works.")));
    expect(result.current.pendingText).toBeUndefined();
    expect(result.current.turns.map((turn) => turn.role)).toEqual([
      "user",
      "assistant",
    ]);

    mockedApi.mockResolvedValueOnce(reply("Sure."));
    await act(() => result.current.send("and Monday?"));
    expect(sentBody(0).conversation_id).toBeUndefined();
    expect(sentBody(1).conversation_id).toBe("c1");
  });

  it("leaves the thread unchanged when a message fails", async () => {
    mockedApi.mockRejectedValueOnce(new Error("Too many messages"));
    const { result } = renderConversation();

    await act(() => result.current.send("hello").catch(() => undefined));

    await waitFor(() =>
      expect(result.current.sendError?.message).toBe("Too many messages"),
    );
    expect(result.current.turns).toEqual([]);
  });

  it("updates an action in the thread once a decision is carried out", async () => {
    const move = {
      id: "a1",
      kind: "move",
      status: "pending",
      event_id: "e1",
      title: "Dentist",
      starts_at: "2026-10-23T10:00:00+02:00",
      ends_at: "2026-10-23T11:00:00+02:00",
      is_all_day: false,
    } as const;
    mockedApi.mockResolvedValueOnce({ ...reply("Waiting."), actions: [move] });
    const { result } = renderConversation();
    await act(() => result.current.send("move the dentist"));

    mockedApi.mockResolvedValueOnce({ ...move, status: "applied" });
    act(() => result.current.decide(move, "apply"));

    await waitFor(() => {
      const answer = result.current.turns[1];
      expect(answer.role === "assistant" && answer.actions[0].status).toBe(
        "applied",
      );
    });
    expect(mockedApi.mock.calls[1][0]).toBe(
      "/api/assistant/conversations/c1/actions/a1",
    );

    mockedApi.mockRejectedValueOnce(new Error("Already decided"));
    act(() => result.current.decide(move, "apply"));
    await waitFor(() =>
      expect(result.current.failedDecision).toEqual({
        actionId: "a1",
        error: new Error("Already decided"),
      }),
    );
  });
});
