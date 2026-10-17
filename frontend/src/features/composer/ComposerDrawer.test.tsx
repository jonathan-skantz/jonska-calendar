import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { ComposerDrawer } from "./ComposerDrawer";
import type { TimelineEntry } from "../../types";

afterEach(cleanup);

describe("ComposerDrawer validation state", () => {
  it("exposes invalid fields to assistive technology", () => {
    const onSave = vi.fn();
    const { container } = render(
      <ComposerDrawer
        type="event"
        editingItem={null}
        selectedDate="2026-10-15"
        draftEventStart={null}
        tags={[]}
        selectedTagIds={[]}
        mainTagId={null}
        timeFormat="browser"
        onSave={onSave}
        onClose={vi.fn()}
        onTypeChange={vi.fn()}
        onSelectedTagIdsChange={vi.fn()}
        onMainTagIdChange={vi.fn()}
      />,
    );
    // An empty title and an end before the start.
    fireEvent.change(screen.getByLabelText("Ends date"), {
      target: { value: "2026-10-14" },
    });
    fireEvent.submit(container.querySelector("form")!);

    expect(onSave).not.toHaveBeenCalled();
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

describe("ComposerDrawer saving", () => {
  const renderTask = (props: { error?: string; pending?: boolean } = {}) => {
    const onSave = vi.fn();
    const { container } = render(
      <ComposerDrawer
        type="task"
        editingItem={null}
        selectedDate="2026-10-15"
        draftEventStart={null}
        tags={[]}
        selectedTagIds={[]}
        mainTagId={null}
        timeFormat="browser"
        onSave={onSave}
        onClose={vi.fn()}
        onTypeChange={vi.fn()}
        onSelectedTagIdsChange={vi.fn()}
        onMainTagIdChange={vi.fn()}
        {...props}
      />,
    );
    return { onSave, form: container.querySelector("form")! };
  };

  it("saves valid form values", () => {
    const { onSave, form } = renderTask();
    fireEvent.change(screen.getByLabelText("Title"), {
      target: { value: "Follow up" },
    });
    fireEvent.submit(form);

    expect(onSave).toHaveBeenCalledOnce();
    expect((onSave.mock.calls[0][0] as FormData).get("title")).toBe(
      "Follow up",
    );
  });

  it("shows the save error and disables saving while pending", () => {
    renderTask({ error: "Unable to create item", pending: true });

    expect(screen.getByRole("alert").textContent).toBe("Unable to create item");
    expect(
      screen
        .getByRole("button", { name: /save item/i })
        .hasAttribute("disabled"),
    ).toBe(true);
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
        onSave={vi.fn()}
        onClose={vi.fn()}
        onTypeChange={vi.fn()}
        onSelectedTagIdsChange={vi.fn()}
        onMainTagIdChange={vi.fn()}
      />,
    );

  it("defaults to Attend and shows skipping only for Attend", () => {
    renderComposer();
    expect(
      screen
        .getByRole("button", { name: "Attend" })
        .getAttribute("aria-pressed"),
    ).toBe("true");
    expect(screen.getByLabelText(/Skipping this one/)).toBeTruthy();

    fireEvent.click(screen.getByRole("button", { name: "Heads-up" }));
    expect(screen.queryByLabelText(/Skipping this one/)).toBeNull();
  });

  it("prefills skipping when editing", () => {
    renderComposer({
      id: "event-1",
      type: "event",
      title: "Lecture",
      kind: "attendable",
      is_skipped: true,
      starts_at: "2026-10-15T10:00:00",
      ends_at: "2026-10-15T12:00:00",
      is_completed: false,
    });
    expect(
      (screen.getByLabelText(/Skipping this one/) as HTMLInputElement).checked,
    ).toBe(true);
  });

  it("shows commute fields only for timed attendable events", () => {
    renderComposer();
    fireEvent.click(screen.getByRole("button", { name: "Heads-up" }));
    expect(screen.queryByLabelText("Commute before (min)")).toBeNull();

    fireEvent.click(screen.getByRole("button", { name: "Attend" }));
    expect(screen.getByLabelText("Commute before (min)")).toBeTruthy();
    expect(screen.getByLabelText("Commute after (min)")).toBeTruthy();

    fireEvent.click(screen.getByLabelText("All day"));
    expect(screen.queryByLabelText("Commute before (min)")).toBeNull();
  });

  it("keeps typed commute values while the fields are hidden", () => {
    renderComposer();
    fireEvent.change(screen.getByLabelText("Commute before (min)"), {
      target: { value: "20" },
    });
    fireEvent.click(screen.getByRole("button", { name: "Heads-up" }));
    fireEvent.click(screen.getByRole("button", { name: "Attend" }));
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

  it("prefills gathering minutes and hides them for informational events", () => {
    renderComposer({
      id: "event-1",
      type: "event",
      title: "Football",
      kind: "attendable",
      starts_at: "2026-10-15T18:00:00",
      ends_at: "2026-10-15T19:30:00",
      gathering_seconds: 1800,
      gathering_is_optional: true,
      gathering_label: "Mingle",
      is_completed: false,
    });
    expect(
      (screen.getByLabelText("Gathering before (min)") as HTMLInputElement)
        .value,
    ).toBe("30");
    expect(
      (screen.getByLabelText("Optional (drop-in)") as HTMLInputElement).checked,
    ).toBe(true);
    expect(
      (screen.getByLabelText("Gathering label") as HTMLInputElement).value,
    ).toBe("Mingle");

    fireEvent.change(screen.getByLabelText("Gathering before (min)"), {
      target: { value: "" },
    });
    expect(screen.queryByLabelText("Gathering label")).toBeNull();
    fireEvent.change(screen.getByLabelText("Gathering before (min)"), {
      target: { value: "15" },
    });
    expect(
      (screen.getByLabelText("Gathering label") as HTMLInputElement).value,
    ).toBe("Mingle");
    fireEvent.click(screen.getByRole("button", { name: "Heads-up" }));
    expect(screen.queryByLabelText("Gathering before (min)")).toBeNull();
  });
});

describe("ComposerDrawer approximate fields", () => {
  const renderComposer = (editingItem: TimelineEntry) =>
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
        onSave={vi.fn()}
        onClose={vi.fn()}
        onTypeChange={vi.fn()}
        onSelectedTagIdsChange={vi.fn()}
        onMainTagIdChange={vi.fn()}
      />,
    );

  it("prefills approximate edges and hides them for all-day events", () => {
    renderComposer({
      id: "event-1",
      type: "event",
      title: "Workshop",
      starts_at: "2026-10-15T10:00:00",
      ends_at: "2026-10-15T18:00:00",
      end_is_approximate: true,
      is_completed: false,
    });
    expect(
      (screen.getByLabelText("Approximate start") as HTMLInputElement).checked,
    ).toBe(false);
    expect(
      (screen.getByLabelText("Approximate end") as HTMLInputElement).checked,
    ).toBe(true);

    fireEvent.click(screen.getByLabelText("All day"));
    expect(screen.queryByLabelText("Approximate end")).toBeNull();

    fireEvent.click(screen.getByLabelText("All day"));
    expect(
      (screen.getByLabelText("Approximate end") as HTMLInputElement).checked,
    ).toBe(true);
  });
});

describe("ComposerDrawer reminders", () => {
  const props = {
    type: "event" as const,
    selectedDate: "2026-10-15",
    draftEventStart: null,
    tags: [],
    selectedTagIds: [],
    mainTagId: null,
    timeFormat: "browser" as const,
    onSave: vi.fn(),
    onClose: vi.fn(),
    onTypeChange: vi.fn(),
    onSelectedTagIdsChange: vi.fn(),
    onMainTagIdChange: vi.fn(),
  };

  it("queues reminders on a new event as form values", () => {
    const { container } = render(
      <ComposerDrawer {...props} editingItem={null} />,
    );
    fireEvent.change(screen.getByLabelText("Add reminder"), {
      target: { value: "900" },
    });
    expect(screen.getByText("15 min before")).toBeTruthy();
    const queued = () =>
      Array.from(
        container.querySelectorAll<HTMLInputElement>(
          'input[name="reminder_offset"]',
        ),
      ).map((input) => input.value);
    expect(queued()).toEqual(["900"]);
    // A preset already in use is not offered again.
    expect(screen.queryByRole("option", { name: "15 min before" })).toBeNull();

    fireEvent.click(
      screen.getByRole("button", { name: "Remove reminder 15 min before" }),
    );
    expect(queued()).toEqual([]);
  });

  it("changes a saved event's reminders right away", () => {
    const onAddReminder = vi.fn();
    const onRemoveReminder = vi.fn();
    render(
      <ComposerDrawer
        {...props}
        editingItem={{
          id: "event-1",
          type: "event",
          title: "Lecture",
          kind: "attendable",
          starts_at: "2026-10-15T10:00:00",
          ends_at: "2026-10-15T12:00:00",
          is_completed: false,
        }}
        reminders={[{ id: "reminder-1", offsetSeconds: 86400 }]}
        onAddReminder={onAddReminder}
        onRemoveReminder={onRemoveReminder}
      />,
    );
    fireEvent.change(
      screen.getByLabelText("Custom reminder (minutes before)"),
      {
        target: { value: "45" },
      },
    );
    fireEvent.click(screen.getByRole("button", { name: "Add" }));
    expect(onAddReminder).toHaveBeenCalledWith(2700);
    fireEvent.click(
      screen.getByRole("button", { name: "Remove reminder 1 day before" }),
    );
    expect(onRemoveReminder).toHaveBeenCalledWith({
      id: "reminder-1",
      offsetSeconds: 86400,
    });
  });
});

describe("ComposerDrawer quick add draft", () => {
  it("prefills a new event from draft values and lists the warnings", () => {
    const { container } = render(
      <ComposerDrawer
        type="event"
        editingItem={null}
        selectedDate="2026-10-15"
        draftEventStart={null}
        tags={[]}
        selectedTagIds={[]}
        mainTagId={null}
        timeFormat="24-hour"
        onSave={vi.fn()}
        onClose={vi.fn()}
        onTypeChange={vi.fn()}
        onSelectedTagIdsChange={vi.fn()}
        onMainTagIdChange={vi.fn()}
        draftValues={{
          title: "Middag",
          starts_at: new Date(2026, 9, 17, 18, 0).toISOString(),
          ends_at: new Date(2026, 9, 17, 21, 0).toISOString(),
          start_is_approximate: true,
          location: "mormor",
          kind: "attendable",
          commute_before_seconds: 1800,
        }}
        draftReminderOffsets={[3600]}
        draftWarnings={['Couldn\'t place "after work"']}
      />,
    );

    expect(screen.getByText("New item")).toBeTruthy();
    expect((screen.getByLabelText("Title") as HTMLInputElement).value).toBe(
      "Middag",
    );
    expect((screen.getByLabelText("Location") as HTMLInputElement).value).toBe(
      "mormor",
    );
    expect(
      (screen.getByLabelText("Approximate start") as HTMLInputElement).checked,
    ).toBe(true);
    expect(
      (screen.getByLabelText("Commute before (min)") as HTMLInputElement).value,
    ).toBe("30");
    expect(
      (screen.getByLabelText("Starts hour") as HTMLInputElement).value,
    ).toBe("18");
    expect((screen.getByLabelText("Ends hour") as HTMLInputElement).value).toBe(
      "21",
    );
    expect(
      container.querySelector<HTMLInputElement>('input[name="reminder_offset"]')
        ?.value,
    ).toBe("3600");
    expect(screen.getByText('Couldn\'t place "after work"')).toBeTruthy();
  });
});
