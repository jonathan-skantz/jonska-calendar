import { cleanup, fireEvent, screen, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { renderWithPreferences } from "../../test/renderWithPreferences";
import type { AssistantAction } from "../../types";
import { AssistantDrawer } from "./AssistantDrawer";
import type {
  AssistantConversation,
  AssistantTurn,
} from "./useAssistantConversation";

const created: AssistantAction = {
  id: "a1",
  kind: "create",
  status: "done",
  event_id: "e1",
  title: "Lunch with Sam",
  starts_at: "2026-10-26T12:00:00+01:00",
  ends_at: "2026-10-26T13:00:00+01:00",
  is_all_day: false,
  warnings: ['No tag named "gym"'],
};

const move: AssistantAction = {
  ...created,
  id: "a2",
  kind: "move",
  status: "pending",
  event_id: "e2",
  title: "Dentist",
  warnings: undefined,
};

const turns: AssistantTurn[] = [
  { role: "user", text: "Lunch with Sam on Monday, and move the dentist" },
  {
    role: "assistant",
    reply: "Lunch is in; the move waits for you.",
    steps: [
      {
        tool: "find_free_slots",
        summary: "Checked Mon 2026-10-26 11:00–14:00: 1 free slots",
      },
      { tool: "create_event", summary: 'Created "Lunch with Sam"' },
    ],
    actions: [created, move],
    ended: false,
  },
];

const renderDrawer = (overrides: Partial<AssistantConversation> = {}) => {
  const conversation: AssistantConversation = {
    turns,
    ended: false,
    pendingText: undefined,
    sendError: null,
    send: vi.fn().mockResolvedValue(undefined),
    deciding: false,
    failedDecision: undefined,
    decide: vi.fn(),
    reset: vi.fn(),
    ...overrides,
  };
  const onOpenEvent = vi.fn();
  renderWithPreferences(
    <AssistantDrawer
      conversation={conversation}
      onOpenEvent={onOpenEvent}
      onClose={vi.fn()}
    />,
  );
  return { ...conversation, onOpenEvent };
};

describe("AssistantDrawer", () => {
  afterEach(cleanup);

  it("shows what the assistant checked and lets the user act on each change", () => {
    const props = renderDrawer();

    expect(
      screen.getByText("Lunch is in; the move waits for you."),
    ).toBeTruthy();
    expect(screen.getByText(/1 free slots/)).toBeTruthy();
    expect(screen.getByText('No tag named "gym"')).toBeTruthy();

    fireEvent.click(screen.getByRole("button", { name: "Open" }));
    expect(props.onOpenEvent).toHaveBeenCalledWith(
      "e1",
      "2026-10-26T12:00:00+01:00",
    );
    fireEvent.click(screen.getByRole("button", { name: "Undo" }));
    expect(props.decide).toHaveBeenCalledWith(created, "undo");
    fireEvent.click(screen.getByRole("button", { name: "Apply" }));
    expect(props.decide).toHaveBeenCalledWith(move, "apply");
  });

  it("shows decided actions without buttons and blocks decisions while one runs", () => {
    renderDrawer({
      turns: [
        {
          role: "assistant",
          reply: "",
          steps: [],
          actions: [{ ...move, status: "applied" }],
          ended: false,
        },
      ],
    });
    expect(screen.getByText("Applied")).toBeTruthy();
    expect(screen.queryByRole("button", { name: "Apply" })).toBeNull();

    cleanup();
    renderDrawer({ deciding: true });
    expect(
      (screen.getByRole("button", { name: "Decline" }) as HTMLButtonElement)
        .disabled,
    ).toBe(true);
  });

  it("sends the trimmed message on Enter and clears the box", () => {
    const props = renderDrawer({ turns: [] });
    const input = screen.getByLabelText(
      "Message the assistant",
    ) as HTMLTextAreaElement;

    fireEvent.change(input, { target: { value: "  what's on Friday?  " } });
    fireEvent.keyDown(input, { key: "Enter" });

    expect(props.send).toHaveBeenCalledWith("what's on Friday?");
    expect(input.value).toBe("");
  });

  it("puts the text back when sending fails", async () => {
    renderDrawer({ send: vi.fn().mockRejectedValue(new Error("Busy")) });
    const input = screen.getByLabelText(
      "Message the assistant",
    ) as HTMLTextAreaElement;

    fireEvent.change(input, { target: { value: "what's on Friday?" } });
    fireEvent.keyDown(input, { key: "Enter" });

    await waitFor(() => expect(input.value).toBe("what's on Friday?"));
  });

  it("shows the waiting message and cannot be reset while answering", () => {
    renderDrawer({ pendingText: "and Saturday?" });

    expect(screen.getByText("and Saturday?")).toBeTruthy();
    expect(screen.getByText("Thinking…")).toBeTruthy();
    expect(
      screen.queryByRole("button", { name: "New conversation" }),
    ).toBeNull();
  });

  it("stops taking messages once the conversation has ended", () => {
    const props = renderDrawer({ ended: true });

    expect(screen.queryByLabelText("Message the assistant")).toBeNull();
    fireEvent.click(screen.getByRole("button", { name: "New conversation" }));
    expect(props.reset).toHaveBeenCalled();
  });

  it("shows message errors under the thread and decision errors on their card", () => {
    renderDrawer({
      sendError: new Error("This conversation has ended, start a new one"),
      failedDecision: {
        actionId: move.id,
        error: new Error("The event no longer exists"),
      },
    });
    const alerts = screen
      .getAllByRole("alert")
      .map((alert) => alert.textContent);
    expect(alerts).toEqual([
      "The event no longer exists",
      "This conversation has ended, start a new one",
    ]);
    expect(
      screen
        .getByText("The event no longer exists")
        .closest(".assistant-action")?.textContent,
    ).toContain("Dentist");
  });
});
