import { act, cleanup, fireEvent, renderHook } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { TimelineEntry } from "../../types";
import { useEntryDrag } from "./useEntryDrag";

const event: TimelineEntry = {
  id: "event-1",
  type: "event",
  title: "Planning",
  kind: "attendable",
  starts_at: new Date(2026, 9, 15, 10, 0).toISOString(),
  ends_at: new Date(2026, 9, 15, 11, 0).toISOString(),
  is_completed: false,
};

// Two day columns, 100px wide and 2400px tall: 100px is one hour.
let item: HTMLElement;
beforeEach(() => {
  document.body.innerHTML = `
    <div class="week-calendar">
      <div class="day-column" data-date="2026-10-15"><article></article></div>
      <div class="day-column" data-date="2026-10-16"></div>
    </div>`;
  document.querySelectorAll<HTMLElement>(".day-column").forEach((column, i) => {
    column.getBoundingClientRect = () => new DOMRect(i * 100, 0, 100, 2400);
  });
  item = document.querySelector("article")!;
});
afterEach(() => {
  cleanup();
  document.body.innerHTML = "";
});

const press = (x: number, y: number, overrides = {}) => ({
  button: 0,
  isPrimary: true,
  pointerType: "mouse",
  clientX: x,
  clientY: y,
  currentTarget: item,
  ...overrides,
});

const pointer = (type: "pointerMove" | "pointerUp", x: number, y: number) =>
  act(() => {
    fireEvent[type](window, { clientX: x, clientY: y });
  });

describe("useEntryDrag", () => {
  it("drags to a snapped start in another column and drops", () => {
    const onDrop = vi.fn();
    const { result } = renderHook(() => useEntryDrag(onDrop));
    // Grabbed 15 minutes into the event.
    act(() => result.current.startDrag(event, press(50, 1025)));

    pointer("pointerMove", 150, 1440);
    expect(result.current.drag).toEqual({
      entryId: "event-1",
      start: new Date(2026, 9, 16, 14, 0),
    });
    expect(document.body.classList.contains("dragging-entry")).toBe(true);

    pointer("pointerUp", 150, 1440);
    expect(onDrop).toHaveBeenCalledWith({
      ...event,
      starts_at: new Date(2026, 9, 16, 14, 0).toISOString(),
      ends_at: new Date(2026, 9, 16, 15, 0).toISOString(),
    });
    expect(result.current.drag).toBeNull();
    expect(document.body.classList.contains("dragging-entry")).toBe(false);
  });

  it("treats a press that barely moves as a click", () => {
    const onDrop = vi.fn();
    const { result } = renderHook(() => useEntryDrag(onDrop));
    act(() => result.current.startDrag(event, press(50, 1025)));

    pointer("pointerMove", 52, 1026);
    expect(result.current.drag).toBeNull();
    pointer("pointerUp", 52, 1026);
    expect(onDrop).not.toHaveBeenCalled();
  });

  it("does not save a drag that ends where it started", () => {
    const onDrop = vi.fn();
    const { result } = renderHook(() => useEntryDrag(onDrop));
    act(() => result.current.startDrag(event, press(50, 1025)));

    pointer("pointerMove", 60, 1035);
    pointer("pointerUp", 60, 1035);
    expect(onDrop).not.toHaveBeenCalled();
  });

  it("cancels on Escape", () => {
    const onDrop = vi.fn();
    const { result } = renderHook(() => useEntryDrag(onDrop));
    act(() => result.current.startDrag(event, press(50, 1025)));
    pointer("pointerMove", 150, 1440);

    act(() => {
      fireEvent.keyDown(window, { key: "Escape" });
    });
    expect(result.current.drag).toBeNull();
    pointer("pointerUp", 150, 1440);
    expect(onDrop).not.toHaveBeenCalled();
  });

  it("ignores shared events, other buttons and touch", () => {
    const { result } = renderHook(() => useEntryDrag(vi.fn()));
    act(() => {
      result.current.startDrag({ ...event, shared_by: "Ana" }, press(50, 1025));
      result.current.startDrag(event, press(50, 1025, { button: 2 }));
      result.current.startDrag(
        event,
        press(50, 1025, { pointerType: "touch" }),
      );
    });
    pointer("pointerMove", 150, 1440);
    expect(result.current.drag).toBeNull();
  });
});
