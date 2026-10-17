import { describe, expect, it } from "vitest";
import type { Reminder } from "../../types";
import {
  describeReminders,
  formatReminderOffset,
  groupReminders,
  isReminderDue,
} from "./reminders";

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

describe("reminder helpers", () => {
  it("formats offsets for timed and all-day events", () => {
    expect(formatReminderOffset(0)).toBe("At start");
    expect(formatReminderOffset(900)).toBe("15 min before");
    expect(formatReminderOffset(3 * 3600)).toBe("3 h before");
    expect(formatReminderOffset(86400)).toBe("1 day before");
    expect(formatReminderOffset(2 * 86400)).toBe("2 days before");
    expect(formatReminderOffset(15 * 3600, true)).toBe("Day before at 09:00");
    expect(formatReminderOffset(86400 + 15 * 3600, true)).toBe(
      "2 days before at 09:00",
    );
    expect(formatReminderOffset(0, true)).toBe("At midnight");
  });

  it("is due once its time has come, until delivered or the event ends", () => {
    const now = new Date("2026-10-15T09:50:00");
    expect(isReminderDue(reminder(), now)).toBe(true);
    expect(isReminderDue(reminder(), new Date("2026-10-15T09:40:00"))).toBe(
      false,
    );
    expect(
      isReminderDue(reminder({ notified_at: "2026-10-15T09:45:00" }), now),
    ).toBe(false);
    expect(isReminderDue(reminder({ event_is_skipped: true }), now)).toBe(
      false,
    );
    expect(isReminderDue(reminder(), new Date("2026-10-15T12:00:00"))).toBe(
      false,
    );
  });

  it("groups reminders by when they fire", () => {
    const groups = groupReminders(
      [
        reminder({ id: "earlier", remind_at: "2026-10-15T08:00:00" }),
        reminder({ id: "today", remind_at: "2026-10-15T18:00:00" }),
        reminder({ id: "tomorrow", remind_at: "2026-10-16T08:00:00" }),
        reminder({ id: "later", remind_at: "2026-10-20T08:00:00" }),
      ],
      new Date("2026-10-15T09:00:00"),
    );
    expect(
      groups.map((group) => [
        group.label,
        group.reminders.map((item) => item.id),
      ]),
    ).toEqual([
      ["Earlier", ["earlier"]],
      ["Today", ["today"]],
      ["Tomorrow", ["tomorrow"]],
      ["Later", ["later"]],
    ]);
  });

  it("describes an event's reminders in offset order", () => {
    expect(
      describeReminders([
        reminder({ offset_seconds: 86400 }),
        reminder({ offset_seconds: 900 }),
      ]),
    ).toBe("Reminders: 15 min before, 1 day before");
  });
});
