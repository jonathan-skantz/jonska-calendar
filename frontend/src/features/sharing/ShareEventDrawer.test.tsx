import {
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
} from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { ShareEventDrawer } from "./ShareEventDrawer";
import type { EventShare, TimelineEntry } from "../../types";

const event: TimelineEntry = {
  id: "event-1",
  type: "event",
  title: "Football",
  location: "Arena",
  starts_at: "2026-10-20T18:00:00",
  ends_at: "2026-10-20T19:30:00",
  is_completed: false,
};

const shares: EventShare[] = [
  {
    id: "share-x",
    recipient_user_id: "x",
    recipient_email: "anna@example.com",
    recipient_display_name: "Anna",
    title: "Title for Anna",
    location: "Gate B",
  },
  {
    id: "share-y",
    recipient_user_id: "y",
    recipient_email: "erik@example.com",
    recipient_display_name: "Erik",
  },
];

const renderDrawer = (props: Partial<Parameters<typeof ShareEventDrawer>[0]>) =>
  render(
    <ShareEventDrawer
      event={event}
      shares={shares}
      onAdd={vi.fn().mockResolvedValue(null)}
      onUpdate={vi.fn().mockResolvedValue(null)}
      onRemove={vi.fn().mockResolvedValue(null)}
      onClose={vi.fn()}
      {...props}
    />,
  );

describe("ShareEventDrawer", () => {
  afterEach(cleanup);

  it("summarizes each recipient's custom fields", () => {
    renderDrawer({});
    expect(screen.getByText("2 custom")).toBeTruthy();
    expect(screen.getByText("Same as event")).toBeTruthy();
  });

  it("shows the event's values as placeholders and clears fields with null", async () => {
    const onUpdate = vi.fn().mockResolvedValue(null);
    renderDrawer({ onUpdate });

    fireEvent.click(screen.getByRole("button", { name: /^Anna/ }));
    const title = screen.getByLabelText("Title") as HTMLInputElement;
    const url = screen.getByLabelText("URL") as HTMLInputElement;
    expect(title.value).toBe("Title for Anna");
    expect(title.placeholder).toBe("Football");
    expect(url.placeholder).toBe("Not set");

    fireEvent.change(title, { target: { value: "  " } });
    fireEvent.change(url, { target: { value: "https://example.com" } });
    fireEvent.click(screen.getByRole("button", { name: "Save for Anna" }));

    await waitFor(() =>
      expect(onUpdate).toHaveBeenCalledWith("share-x", {
        title: null,
        location: "Gate B",
        url: "https://example.com",
        description: null,
      }),
    );
  });

  it("adds a recipient and shows errors from the server", async () => {
    const onAdd = vi
      .fn()
      .mockResolvedValueOnce("recipient not found")
      .mockResolvedValueOnce(null);
    renderDrawer({ onAdd });
    const email = screen.getByLabelText("Add recipient");

    fireEvent.change(email, { target: { value: " Nobody@Example.com" } });
    fireEvent.click(screen.getByRole("button", { name: "Add" }));
    expect((await screen.findByRole("alert")).textContent).toBe(
      "recipient not found",
    );
    expect(onAdd).toHaveBeenCalledWith("nobody@example.com");

    fireEvent.change(email, { target: { value: "erik@example.com" } });
    fireEvent.click(screen.getByRole("button", { name: "Add" }));
    await waitFor(() => expect(screen.queryByRole("alert")).toBeNull());
    expect(
      screen
        .getByRole("button", { name: /^Erik/ })
        .getAttribute("aria-expanded"),
    ).toBe("true");
  });
});
