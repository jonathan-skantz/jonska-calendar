import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { ComposerDrawer } from "./ComposerDrawer";
import type { TimelineEntry } from "../types";

afterEach(cleanup);

describe("ComposerDrawer validation state", () => {
  it("exposes invalid fields to assistive technology", () => {
    render(
      <ComposerDrawer
        type="event"
        editingItem={null}
        selectedDate="2026-10-15"
        draftEventStart={null}
        tags={[]}
        selectedTagIds={[]}
        mainTagId={null}
        timeFormat="browser"
        fieldErrors={{ title: true, starts_at: true, ends_at: true }}
        error="Please correct the highlighted fields."
        onSubmit={vi.fn()}
        onClose={vi.fn()}
        onTypeChange={vi.fn()}
        onSelectedTagIdsChange={vi.fn()}
        onMainTagIdChange={vi.fn()}
      />,
    );

    expect(screen.getByLabelText("Title").getAttribute("aria-invalid")).toBe(
      "true",
    );
    expect(screen.getByRole("alert").textContent).toBe(
      "Please correct the highlighted fields.",
    );
    expect(
      screen.getByLabelText("Title").getAttribute("aria-describedby"),
    ).toBe("composer-error");
    expect(
      screen.getByLabelText("Starts date").getAttribute("aria-invalid"),
    ).toBe("true");
    expect(
      screen.getByLabelText("Starts hour").getAttribute("aria-invalid"),
    ).toBe("true");
    expect(
      screen.getByLabelText("Starts minute").getAttribute("aria-describedby"),
    ).toBe("composer-error");
    expect(
      screen.getByLabelText("Ends date").getAttribute("aria-invalid"),
    ).toBe("true");
    expect(
      screen.getByLabelText("Description").getAttribute("aria-invalid"),
    ).toBe("false");
  });
});

describe("ComposerDrawer commute fields", () => {
  const renderComposer = (editingItem: TimelineEntry | null = null) =>
    render(
      <ComposerDrawer
        type="event"
        editingItem={editingItem}
        selectedDate="2026-10-15"
        draftEventStart={null}
        tags={[]}
        selectedTagIds={[]}
        mainTagId={null}
        timeFormat="browser"
        fieldErrors={{}}
        onSubmit={vi.fn()}
        onClose={vi.fn()}
        onTypeChange={vi.fn()}
        onSelectedTagIdsChange={vi.fn()}
        onMainTagIdChange={vi.fn()}
      />,
    );

  it("shows commute fields only for timed attendable events", () => {
    const { container } = renderComposer();
    expect(screen.queryByLabelText("Commute before (min)")).toBeNull();

    fireEvent.change(container.querySelector('select[name="kind"]')!, {
      target: { value: "attendable" },
    });
    expect(screen.getByLabelText("Commute before (min)")).toBeTruthy();
    expect(screen.getByLabelText("Commute after (min)")).toBeTruthy();

    fireEvent.click(screen.getByLabelText("All day"));
    expect(screen.queryByLabelText("Commute before (min)")).toBeNull();
  });

  it("keeps typed commute values while the fields are hidden", () => {
    const { container } = renderComposer();
    const kind = container.querySelector('select[name="kind"]')!;
    fireEvent.change(kind, { target: { value: "attendable" } });
    fireEvent.change(screen.getByLabelText("Commute before (min)"), {
      target: { value: "20" },
    });
    fireEvent.change(kind, { target: { value: "informational" } });
    fireEvent.change(kind, { target: { value: "attendable" } });
    expect(
      (screen.getByLabelText("Commute before (min)") as HTMLInputElement).value,
    ).toBe("20");
  });

  it("prefills commute minutes when editing", () => {
    renderComposer({
      id: "event-1",
      type: "event",
      title: "Football",
      kind: "attendable",
      starts_at: "2026-10-15T18:00:00",
      ends_at: "2026-10-15T19:30:00",
      commute_before_seconds: 1200,
      is_completed: false,
    });
    expect(
      (screen.getByLabelText("Commute before (min)") as HTMLInputElement).value,
    ).toBe("20");
    expect(
      (screen.getByLabelText("Commute after (min)") as HTMLInputElement).value,
    ).toBe("");
  });
});
