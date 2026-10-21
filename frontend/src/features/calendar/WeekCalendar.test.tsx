import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { WeekCalendar } from "./WeekCalendar";

const renderCalendar = (onSwipe: (direction: 1 | -1) => void) =>
  render(
    <WeekCalendar
      dates={[new Date(2026, 9, 15), new Date(2026, 9, 16)]}
      today={new Date(2026, 9, 15, 12)}
      onDayClick={vi.fn()}
      renderDay={() => null}
      renderAllDay={() => null}
      transitCardPills={null}
      hasAllDayEvents={false}
      weatherByDay={new Map()}
      scrollToHour={7}
      onSwipe={onSwipe}
    />,
  );

const swipeLeft = (calendar: HTMLElement) => {
  fireEvent.pointerDown(calendar, {
    pointerType: "touch",
    clientX: 300,
    clientY: 400,
  });
  fireEvent.pointerUp(calendar, {
    pointerType: "touch",
    clientX: 200,
    clientY: 410,
  });
};

describe("WeekCalendar swipe", () => {
  afterEach(cleanup);

  it("moves the range on a horizontal swipe", () => {
    const onSwipe = vi.fn();
    renderCalendar(onSwipe);
    swipeLeft(screen.getByLabelText("Week calendar"));
    expect(onSwipe).toHaveBeenCalledWith(1);
  });

  it("ignores a pointerup claimed by a drag", () => {
    const onSwipe = vi.fn();
    renderCalendar(onSwipe);
    // What useEntryDrag does when a drag ends.
    const claim = (event: Event) => event.preventDefault();
    window.addEventListener("pointerup", claim, true);
    swipeLeft(screen.getByLabelText("Week calendar"));
    window.removeEventListener("pointerup", claim, true);
    expect(onSwipe).not.toHaveBeenCalled();
  });
});
