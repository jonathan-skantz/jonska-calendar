import { act, renderHook } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { PHONE_DAY_COUNT, WEEK_DAY_COUNT, dateKey } from "../../lib/calendar";
import {
  getRangeAnchor,
  getRangeStart,
  useCalendarState,
} from "./useCalendarState";

// Friday, 2026-10-16.
const today = new Date(2026, 9, 16, 14, 30);

describe("getRangeStart", () => {
  it("snaps a full week to the calendar week start", () => {
    expect(dateKey(getRangeStart(today, 1, WEEK_DAY_COUNT))).toBe("2026-10-12");
  });

  it("starts shorter ranges at the anchor day", () => {
    expect(dateKey(getRangeStart(today, 1, PHONE_DAY_COUNT))).toBe(
      "2026-10-16",
    );
  });
});

describe("getRangeAnchor", () => {
  it("keeps today when it is visible", () => {
    const rangeStart = new Date(2026, 9, 12);
    expect(getRangeAnchor({ rangeStart, dayCount: 7 }, today)).toBe(today);
  });

  it("falls back to the middle of the range", () => {
    const rangeStart = new Date(2026, 10, 2);
    expect(dateKey(getRangeAnchor({ rangeStart, dayCount: 7 }, today))).toBe(
      "2026-11-05",
    );
  });
});

describe("useCalendarState", () => {
  it("goes to the week of a date", () => {
    const { result } = renderHook(() =>
      useCalendarState(today, 1, WEEK_DAY_COUNT),
    );
    act(() => result.current.goToDate(new Date(2026, 11, 3, 9)));
    expect(dateKey(result.current.rangeStart)).toBe("2026-11-30");
    expect(result.current.selectedDate).toBe("2026-12-03");
  });

  it("steps by the visible day count", () => {
    const { result } = renderHook(() =>
      useCalendarState(today, 1, PHONE_DAY_COUNT),
    );
    act(() => result.current.moveRange(1));
    expect(dateKey(result.current.rangeStart)).toBe("2026-10-19");
    act(() => result.current.moveRange(-1));
    act(() => result.current.moveRange(-1));
    expect(dateKey(result.current.rangeStart)).toBe("2026-10-13");
  });

  it("snaps back to a week when switching from the phone range", () => {
    const { result, rerender } = renderHook(
      ({ dayCount }) => useCalendarState(today, 1, dayCount),
      { initialProps: { dayCount: PHONE_DAY_COUNT } },
    );
    act(() => result.current.moveRange(1));
    act(() => result.current.moveRange(1));
    // 2026-10-22 .. 24 is shown; today is not, so the middle (10-23) anchors.
    rerender({ dayCount: WEEK_DAY_COUNT });
    expect(dateKey(result.current.rangeStart)).toBe("2026-10-19");
  });

  it("jumps back to the range containing today", () => {
    const { result } = renderHook(() =>
      useCalendarState(today, 1, WEEK_DAY_COUNT),
    );
    act(() => result.current.moveRange(1));
    act(() => result.current.moveRange(1));
    act(() => result.current.goToToday());
    expect(dateKey(result.current.rangeStart)).toBe("2026-10-12");
    expect(result.current.selectedDate).toBe("2026-10-16");
  });
});
