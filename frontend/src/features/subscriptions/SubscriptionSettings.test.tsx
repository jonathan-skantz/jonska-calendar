import {
  cleanup,
  fireEvent,
  render,
  screen,
  within,
} from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import type { Subscription } from "../../types";
import { SubscriptionSettings } from "./SubscriptionSettings";

afterEach(() => {
  cleanup();
  vi.restoreAllMocks();
});

const school: Subscription = {
  id: "sub-1",
  name: "School",
  color: "#336699",
  url: "https://example.com/school.ics",
  last_fetched_at: "2026-10-22T08:00:00Z",
  last_success_at: "2026-10-22T08:00:00Z",
};

const renderSettings = (
  subscriptions: Subscription[],
  handlers: Partial<Parameters<typeof SubscriptionSettings>[0]> = {},
) => {
  const props = {
    subscriptions,
    onAdd: vi.fn().mockResolvedValue(null),
    onUpdate: vi.fn().mockResolvedValue(null),
    onRefresh: vi.fn().mockResolvedValue(null),
    onRemove: vi.fn().mockResolvedValue(null),
    ...handlers,
  };
  render(<SubscriptionSettings {...props} />);
  return props;
};

describe("SubscriptionSettings", () => {
  it("subscribes with the address, name and color", async () => {
    const { onAdd } = renderSettings([]);
    fireEvent.change(screen.getByLabelText("Calendar address"), {
      target: { value: "webcal://example.com/holidays.ics" },
    });
    fireEvent.change(screen.getByLabelText("Name (optional)"), {
      target: { value: "Holidays" },
    });
    fireEvent.click(screen.getByRole("button", { name: "Subscribe" }));

    await screen.findByText("Subscribed.");
    expect(onAdd).toHaveBeenCalledWith("webcal://example.com/holidays.ics", {
      name: "Holidays",
      color: "#547968",
    });
  });

  it("shows why subscribing failed", async () => {
    renderSettings([], {
      onAdd: vi.fn().mockResolvedValue("the calendar could not be reached"),
    });
    fireEvent.change(screen.getByLabelText("Calendar address"), {
      target: { value: "https://example.com/missing.ics" },
    });
    fireEvent.click(screen.getByRole("button", { name: "Subscribe" }));
    await screen.findByText("the calendar could not be reached");
  });

  it("shows a failed refresh and keeps the last events", () => {
    renderSettings([
      {
        ...school,
        last_fetched_at: "2026-10-22T14:00:00Z",
        last_error: "the calendar's server answered 410 Gone",
      },
    ]);
    expect(
      screen.getByText(
        /Last refresh failed .*: the calendar's server answered 410 Gone\. Showing events from/,
      ),
    ).toBeTruthy();
  });

  it("renames, refreshes and unsubscribes", async () => {
    vi.spyOn(window, "confirm").mockReturnValue(true);
    const { onUpdate, onRefresh, onRemove } = renderSettings([school]);
    const item = within(screen.getByRole("list", { name: "Subscriptions" }));

    fireEvent.change(item.getByLabelText("Name of School"), {
      target: { value: "Uni" },
    });
    fireEvent.click(item.getByRole("button", { name: "Save" }));
    await screen.findByText("Saved.");
    expect(onUpdate).toHaveBeenCalledWith("sub-1", {
      name: "Uni",
      color: "#336699",
    });

    fireEvent.click(item.getByRole("button", { name: "Refresh now" }));
    await screen.findByText("Refreshed.");
    expect(onRefresh).toHaveBeenCalledWith("sub-1");

    fireEvent.click(item.getByRole("button", { name: "Unsubscribe" }));
    await screen.findByText("Unsubscribed.");
    expect(onRemove).toHaveBeenCalledWith("sub-1");
  });
});
