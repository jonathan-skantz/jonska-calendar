import { cleanup, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { SharedEventDrawer } from "./SharedEventDrawer";
import type { TimelineEntry } from "../types";

const entry: TimelineEntry = {
  id: "event-1",
  type: "event",
  title: "Title for Anna",
  location: "Gate B",
  url: "javascript:alert(1)",
  description: "Bring boots",
  starts_at: "2026-10-20T18:00:00",
  ends_at: "2026-10-20T19:30:00",
  is_shared: true,
  shared_by: "Jonathan",
  is_completed: false,
};

describe("SharedEventDrawer", () => {
  afterEach(cleanup);

  it("shows the shared fields read-only", () => {
    render(
      <SharedEventDrawer
        entry={entry}
        timeFormat="24-hour"
        onClose={vi.fn()}
      />,
    );

    expect(screen.getByText("Shared by Jonathan")).toBeTruthy();
    expect(screen.getByText("Title for Anna")).toBeTruthy();
    expect(screen.getByText("Tue, Oct 20 18:00 – 19:30")).toBeTruthy();
    expect(screen.getByText("Gate B")).toBeTruthy();
    expect(
      screen.getByText("Only Jonathan can change this event."),
    ).toBeTruthy();
    expect(screen.queryByRole("textbox")).toBeNull();
    // Only web links are clickable.
    expect(screen.queryByRole("link")).toBeNull();
  });

  it("marks approximate edges", () => {
    render(
      <SharedEventDrawer
        entry={{ ...entry, start_is_approximate: true }}
        timeFormat="24-hour"
        onClose={vi.fn()}
      />,
    );

    expect(screen.getByText("Tue, Oct 20 ~18:00 – 19:30")).toBeTruthy();
  });
});
