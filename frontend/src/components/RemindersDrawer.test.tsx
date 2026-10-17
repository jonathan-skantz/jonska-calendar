import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { RemindersDrawer } from "./RemindersDrawer";
import type { Reminder } from "../types";

const reminder = (overrides: Partial<Reminder> = {}): Reminder => ({
  id: "reminder-1",
  event_id: "event-1",
  offset_seconds: 900,
  remind_at: "2026-10-15T09:45:00",
  event_title: "Lecture",
  event_starts_at: "2026-10-15T10:00:00",
  event_ends_at: "2026-10-15T12:00:00",
  ...overrides,
});

describe("RemindersDrawer", () => {
  afterEach(cleanup);

  it("lists reminders by day and lets the user open or remove them", () => {
    const onOpenEvent = vi.fn();
    const onRemove = vi.fn();
    const birthday = reminder({
      id: "birthday",
      event_id: "event-2",
      event_title: "Anna's birthday",
      offset_seconds: 15 * 3600,
      remind_at: "2026-10-16T09:00:00",
      event_is_all_day: true,
      is_shared: true,
    });
    render(
      <RemindersDrawer
        reminders={[reminder(), birthday]}
        now={new Date("2026-10-15T09:00:00")}
        timeFormat="24-hour"
        onOpenEvent={onOpenEvent}
        onRemove={onRemove}
        onClose={vi.fn()}
      />,
    );

    expect(
      screen.getAllByRole("heading", { level: 3 }).map((h) => h.textContent),
    ).toEqual(["Today", "Tomorrow"]);
    expect(screen.getByText("15 min before")).toBeTruthy();
    expect(
      screen.getByText("Day before at 09:00 · shared with you"),
    ).toBeTruthy();

    fireEvent.click(screen.getByText("Lecture"));
    expect(onOpenEvent).toHaveBeenCalledWith("event-1");
    fireEvent.click(
      screen.getByRole("button", {
        name: "Remove reminder for Anna's birthday",
      }),
    );
    expect(onRemove).toHaveBeenCalledWith(birthday);
  });

  it("explains where reminders come from when there are none", () => {
    render(
      <RemindersDrawer
        reminders={[]}
        now={new Date()}
        timeFormat="24-hour"
        onOpenEvent={vi.fn()}
        onRemove={vi.fn()}
        onClose={vi.fn()}
      />,
    );
    expect(screen.getByText(/No reminders/)).toBeTruthy();
  });
});
