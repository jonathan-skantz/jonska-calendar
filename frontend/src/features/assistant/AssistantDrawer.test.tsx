import { cleanup, fireEvent, screen, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { renderWithPreferences } from "../../test/renderWithPreferences";
import type { AssistantProposal } from "../../types";
import { AssistantDrawer } from "./AssistantDrawer";
import type { AssistantTurn } from "./useAssistantConversation";

const proposal: AssistantProposal = {
  title: "Lunch with Sam",
  starts_at: "2026-10-26T12:00:00+01:00",
  ends_at: "2026-10-26T13:00:00+01:00",
  is_all_day: false,
  warnings: ['No tag named "gym"'],
};

const turns: AssistantTurn[] = [
  { role: "user", text: "Lunch with Sam on Friday, or Monday if I'm busy" },
  {
    role: "assistant",
    reply: "Friday is taken, so I suggested Monday.",
    steps: [
      {
        tool: "find_free_slots",
        summary: "Checked Fri 2026-10-23 11:00–14:00: no free time",
      },
      { tool: "propose_event", summary: 'Proposed "Lunch with Sam"' },
    ],
    proposals: [proposal],
    ended: false,
  },
];

const renderDrawer = (
  overrides: Partial<Parameters<typeof AssistantDrawer>[0]> = {},
) => {
  const props = {
    turns,
    ended: false,
    pendingText: undefined,
    error: "",
    onSend: vi.fn().mockResolvedValue(undefined),
    onReset: vi.fn(),
    onOpenProposal: vi.fn(),
    onClose: vi.fn(),
    ...overrides,
  };
  renderWithPreferences(<AssistantDrawer {...props} />);
  return props;
};

describe("AssistantDrawer", () => {
  afterEach(cleanup);

  it("shows what the assistant checked and opens its proposal in the form", () => {
    const props = renderDrawer();

    expect(
      screen.getByText("Friday is taken, so I suggested Monday."),
    ).toBeTruthy();
    expect(screen.getByText(/no free time/)).toBeTruthy();
    expect(screen.getByText('No tag named "gym"')).toBeTruthy();

    fireEvent.click(screen.getByRole("button", { name: "Open in form" }));
    expect(props.onOpenProposal).toHaveBeenCalledWith(proposal);
  });

  it("sends the trimmed message on Enter and clears the box", () => {
    const props = renderDrawer({ turns: [] });
    const input = screen.getByLabelText(
      "Message the assistant",
    ) as HTMLTextAreaElement;

    fireEvent.change(input, { target: { value: "  what's on Friday?  " } });
    fireEvent.keyDown(input, { key: "Enter" });

    expect(props.onSend).toHaveBeenCalledWith("what's on Friday?");
    expect(input.value).toBe("");
  });

  it("puts the text back when sending fails", async () => {
    renderDrawer({ onSend: vi.fn().mockRejectedValue(new Error("Busy")) });
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
    expect(props.onReset).toHaveBeenCalled();
  });

  it("shows request errors", () => {
    renderDrawer({ error: "This conversation has ended, start a new one" });
    expect(screen.getByRole("alert").textContent).toBe(
      "This conversation has ended, start a new one",
    );
  });
});
