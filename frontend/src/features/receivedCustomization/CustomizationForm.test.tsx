import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { CustomizationForm } from "./CustomizationForm";
import type { TimelineEntry } from "../../types";

const entry: TimelineEntry = {
  id: "event-1",
  type: "event",
  title: "My lecture",
  location: "Hall A",
  starts_at: "2026-10-20T09:15:00Z",
  ends_at: "2026-10-20T10:00:00Z",
  is_shared: true,
  shared_by: "Jonathan",
  is_completed: false,
  customization: { title: "My lecture", start_shift_seconds: 900 },
  original: {
    title: "Lecture",
    location: "Hall A",
    starts_at: "2026-10-20T09:00:00Z",
    ends_at: "2026-10-20T10:00:00Z",
  },
};

describe("CustomizationForm", () => {
  afterEach(cleanup);

  it("starts from the customization and shows received values as placeholders", () => {
    render(
      <CustomizationForm entry={entry} onSave={vi.fn()} onCancel={vi.fn()} />,
    );

    expect(screen.getByLabelText("Title")).toHaveProperty(
      "value",
      "My lecture",
    );
    expect(screen.getByLabelText("Title")).toHaveProperty(
      "placeholder",
      "Lecture",
    );
    expect(screen.getByLabelText("Location")).toHaveProperty(
      "placeholder",
      "Hall A",
    );
    expect(screen.getByLabelText("Move start (min)")).toHaveProperty(
      "value",
      "15",
    );
  });

  it("saves the trimmed customization and closes", async () => {
    const onSave = vi.fn().mockResolvedValue(null);
    const onCancel = vi.fn();
    render(
      <CustomizationForm entry={entry} onSave={onSave} onCancel={onCancel} />,
    );

    fireEvent.change(screen.getByLabelText("My note"), {
      target: { value: " Zoom link " },
    });
    fireEvent.change(screen.getByLabelText("Move end (min)"), {
      target: { value: "-30" },
    });
    fireEvent.click(screen.getByText("Save my version"));

    await vi.waitFor(() => expect(onCancel).toHaveBeenCalled());
    expect(onSave).toHaveBeenCalledWith({
      title: "My lecture",
      note: "Zoom link",
      start_shift_seconds: 900,
      end_shift_seconds: -1800,
    });
  });

  it("keeps the form open with the server's error", async () => {
    const onCancel = vi.fn();
    render(
      <CustomizationForm
        entry={entry}
        onSave={vi
          .fn()
          .mockResolvedValue("start and end shifts must be whole minutes")}
        onCancel={onCancel}
      />,
    );

    fireEvent.click(screen.getByText("Save my version"));

    expect(
      await screen.findByText("start and end shifts must be whole minutes"),
    ).toBeTruthy();
    expect(onCancel).not.toHaveBeenCalled();
  });

  it("does not offer moving the times of an all-day event", () => {
    render(
      <CustomizationForm
        entry={{ ...entry, is_all_day: true }}
        onSave={vi.fn()}
        onCancel={vi.fn()}
      />,
    );

    expect(screen.queryByLabelText("Move start (min)")).toBeNull();
  });
});
