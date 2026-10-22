import { cleanup, fireEvent, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { SharedEventDrawer } from "./SharedEventDrawer";
import type { TimelineEntry } from "../../types";
import { renderWithPreferences } from "../../test/renderWithPreferences";

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
    renderWithPreferences(
      <SharedEventDrawer entry={entry} onClose={vi.fn()} />,
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

  it("shows a subscribed event read-only without reminders", () => {
    const { is_shared: _shared, shared_by: _sharedBy, ...rest } = entry;
    renderWithPreferences(
      <SharedEventDrawer
        entry={{
          ...rest,
          subscription: { id: "sub-1", name: "School", color: "#123456" },
        }}
        onClose={vi.fn()}
      />,
    );

    expect(screen.getByText("From School")).toBeTruthy();
    expect(
      screen.getByText(
        "This event comes from a subscribed calendar and changes when the calendar does.",
      ),
    ).toBeTruthy();
    expect(screen.queryByText(/Remind/)).toBeNull();
  });

  it("marks approximate edges", () => {
    renderWithPreferences(
      <SharedEventDrawer
        entry={{ ...entry, start_is_approximate: true }}
        onClose={vi.fn()}
      />,
    );

    expect(screen.getByText("Tue, Oct 20 ~18:00 – 19:30")).toBeTruthy();
  });

  it("shows gathering time", () => {
    renderWithPreferences(
      <SharedEventDrawer
        entry={{
          ...entry,
          kind: "attendable",
          gathering_seconds: 1800,
          gathering_is_optional: true,
          gathering_label: "Mingle",
        }}
        onClose={vi.fn()}
      />,
    );

    expect(
      screen.getByText("Tue, Oct 20 18:00 – 19:30, Mingle 17:30 (optional)"),
    ).toBeTruthy();
  });

  it("shows the recipient's own version with the original on demand", () => {
    const onCustomize = vi.fn();
    const onReset = vi.fn();
    renderWithPreferences(
      <SharedEventDrawer
        entry={{
          ...entry,
          title: "My lecture",
          starts_at: "2026-10-20T18:15:00",
          customization: { title: "My lecture", note: "Zoom link" },
          original: {
            title: "Title for Anna",
            location: "Gate B",
            starts_at: "2026-10-20T18:00:00",
            ends_at: "2026-10-20T19:30:00",
          },
        }}
        onClose={vi.fn()}
        onCustomize={onCustomize}
        onResetCustomization={onReset}
      />,
    );

    expect(screen.getByText("My lecture")).toBeTruthy();
    expect(screen.getByText("Zoom link")).toBeTruthy();
    expect(screen.queryByLabelText("Original")).toBeNull();
    fireEvent.click(screen.getByText("Show original"));
    expect(screen.getByLabelText("Original").textContent).toContain(
      "Title for AnnaWhenTue, Oct 20 18:00 – 19:30",
    );

    fireEvent.click(screen.getByText("Edit my version"));
    fireEvent.click(screen.getByText("Reset to original"));
    expect(onCustomize).toHaveBeenCalled();
    expect(onReset).toHaveBeenCalled();
  });

  it("shows the recipient's own commute and the original gathering", () => {
    renderWithPreferences(
      <SharedEventDrawer
        entry={{
          ...entry,
          kind: "attendable",
          commute_before_seconds: 1200,
          commute_after_seconds: 600,
          customization: { commute_before_seconds: 1200 },
          original: {
            title: "Title for Anna",
            starts_at: "2026-10-20T18:00:00",
            ends_at: "2026-10-20T19:30:00",
            gathering_seconds: 1800,
          },
        }}
        onClose={vi.fn()}
      />,
    );

    expect(screen.getByText("20 min before, 10 min after")).toBeTruthy();
    fireEvent.click(screen.getByText("Show original"));
    expect(screen.getByLabelText("Original").textContent).toContain(
      "Tue, Oct 20 18:00 – 19:30, gather 17:30",
    );
  });

  it("offers to customize an event not changed yet", () => {
    renderWithPreferences(
      <SharedEventDrawer
        entry={entry}
        onClose={vi.fn()}
        onCustomize={vi.fn()}
        onResetCustomization={vi.fn()}
      />,
    );

    expect(screen.getByText("Customize")).toBeTruthy();
    expect(screen.queryByText("Reset to original")).toBeNull();
    expect(screen.queryByText(/You changed this event/)).toBeNull();
  });

  it("shows the editor in place of the details", () => {
    renderWithPreferences(
      <SharedEventDrawer
        entry={entry}
        onClose={vi.fn()}
        onCustomize={vi.fn()}
        editor={<p>Editing</p>}
      />,
    );

    expect(screen.getByText("Editing")).toBeTruthy();
    expect(screen.queryByText("Gate B")).toBeNull();
    expect(screen.queryByText("Customize")).toBeNull();
  });
});
