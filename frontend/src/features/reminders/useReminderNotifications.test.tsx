import { act, cleanup, renderHook } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { Reminder } from "../../types";
import { useReminderNotifications } from "./useReminderNotifications";

const reminder = (overrides: Partial<Reminder> = {}): Reminder => ({
  id: "reminder-1",
  event_id: "event-1",
  offset_seconds: 900,
  remind_at: "2026-10-15T09:45:00",
  event_title: "Lecture",
  event_location: "Hall B",
  event_starts_at: "2026-10-15T10:00:00",
  occurrence_start: "2026-10-15T10:00:00",
  event_ends_at: "2026-10-15T12:00:00",
  ...overrides,
});

const notifications: { title: string; options?: NotificationOptions }[] = [];
class FakeNotification {
  static permission: NotificationPermission = "granted";
  constructor(title: string, options?: NotificationOptions) {
    notifications.push({ title, options });
  }
}

describe("useReminderNotifications", () => {
  beforeEach(() => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date("2026-10-15T09:30:00"));
    notifications.length = 0;
    FakeNotification.permission = "granted";
    vi.stubGlobal("Notification", FakeNotification);
  });
  afterEach(() => {
    cleanup();
    vi.useRealTimers();
    vi.unstubAllGlobals();
  });

  it("notifies when a reminder is due and records the delivery once", async () => {
    const markNotified = vi.fn().mockResolvedValue(null);
    const { result, rerender } = renderHook(
      ({ reminders }) =>
        useReminderNotifications(reminders, markNotified, "24-hour"),
      { initialProps: { reminders: [reminder()] } },
    );
    expect(notifications).toHaveLength(0);

    await act(async () => {
      vi.advanceTimersByTime(15 * 60 * 1000);
    });
    expect(notifications).toEqual([
      {
        title: "Lecture",
        options: {
          body: "10:00 · Hall B",
          tag: "reminder-1@2026-10-15T10:00:00",
        },
      },
    ]);
    expect(markNotified).toHaveBeenCalledWith(
      expect.objectContaining({ id: "reminder-1" }),
    );
    expect(result.current.toasts).toEqual([]);

    // A refetch before the server reflects the delivery must not repeat it.
    rerender({ reminders: [reminder()] });
    expect(notifications).toHaveLength(1);
    expect(markNotified).toHaveBeenCalledOnce();
  });

  it("shows reminders missed while the app was closed as toasts", () => {
    vi.setSystemTime(new Date("2026-10-15T09:50:00"));
    const markNotified = vi.fn().mockResolvedValue(null);
    const { result } = renderHook(() =>
      useReminderNotifications(
        [reminder(), reminder({ id: "skipped", event_is_skipped: true })],
        markNotified,
        "24-hour",
      ),
    );
    expect(notifications).toHaveLength(0);
    expect(result.current.toasts).toEqual([
      { reminder: reminder(), missed: true },
    ]);
    expect(markNotified).toHaveBeenCalledExactlyOnceWith(
      expect.objectContaining({ id: "reminder-1" }),
    );
  });

  it("falls back to a toast without notification permission", async () => {
    FakeNotification.permission = "denied";
    const { result } = renderHook(() =>
      useReminderNotifications(
        [reminder()],
        vi.fn().mockResolvedValue(null),
        "24-hour",
      ),
    );
    await act(async () => {
      vi.advanceTimersByTime(15 * 60 * 1000);
    });
    expect(notifications).toHaveLength(0);
    expect(result.current.toasts).toEqual([
      { reminder: reminder(), missed: false },
    ]);

    act(() => result.current.dismiss("reminder-1@2026-10-15T10:00:00"));
    expect(result.current.toasts).toEqual([]);
  });

  it("delivers a repeating event's reminder once for each occurrence", async () => {
    const markNotified = vi.fn().mockResolvedValue(null);
    const tomorrow = reminder({
      remind_at: "2026-10-16T09:45:00",
      event_starts_at: "2026-10-16T10:00:00",
      event_ends_at: "2026-10-16T12:00:00",
      occurrence_start: "2026-10-16T10:00:00",
    });
    renderHook(() =>
      useReminderNotifications([reminder(), tomorrow], markNotified, "24-hour"),
    );
    await act(async () => {
      vi.advanceTimersByTime(24 * 60 * 60 * 1000 + 15 * 60 * 1000);
    });
    expect(notifications.map(({ options }) => options?.tag)).toEqual([
      "reminder-1@2026-10-15T10:00:00",
      "reminder-1@2026-10-16T10:00:00",
    ]);
    expect(markNotified).toHaveBeenLastCalledWith(tomorrow);
  });
});
