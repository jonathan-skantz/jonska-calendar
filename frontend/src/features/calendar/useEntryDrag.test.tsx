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
  target: item,
  ...overrides,
});

const pointer = (type: "pointerMove" | "pointerUp", x: number, y: number) =>
  act(() => {
    fireEvent[type](window, { clientX: x, clientY: y });
  });

describe("useEntryDrag", () => {
  it("drags to a snapped start in another column and drops", async () => {
    let saved: () => void = () => {};
    const onDrop = vi.fn(
      () => new Promise<void>((resolve) => (saved = resolve)),
    );
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
    expect(document.body.classList.contains("dragging-entry")).toBe(false);
    // Stays at its new place until the save settles.
    expect(result.current.drag).toEqual({
      entryId: "event-1",
      start: new Date(2026, 9, 16, 14, 0),
      dropped: true,
    });
    await act(async () => saved());
    expect(result.current.drag).toBeNull();
  });

  it("lets a failed save put the item back", async () => {
    let failed: () => void = () => {};
    const onDrop = vi.fn(
      () => new Promise<void>((_, reject) => (failed = () => reject())),
    );
    const { result } = renderHook(() => useEntryDrag(onDrop));
    act(() => result.current.startDrag(event, press(50, 1025)));
    pointer("pointerMove", 150, 1440);
    pointer("pointerUp", 150, 1440);
    expect(result.current.drag?.dropped).toBe(true);

    await act(async () => failed());
    expect(result.current.drag).toBeNull();
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

  it("ignores shared events and other buttons", () => {
    const { result } = renderHook(() => useEntryDrag(vi.fn()));
    act(() => {
      result.current.startDrag({ ...event, shared_by: "Ana" }, press(50, 1025));
      result.current.startDrag(event, press(50, 1025, { button: 2 }));
    });
    pointer("pointerMove", 150, 1440);
    expect(result.current.drag).toBeNull();
  });
});

describe("useEntryDrag on touch screens", () => {
  beforeEach(() => vi.useFakeTimers());
  afterEach(() => vi.useRealTimers());

  const touch = { pointerType: "touch" };

  it("picks an item up after a long press and drops it where it is moved", () => {
    const onDrop = vi.fn();
    const { result } = renderHook(() => useEntryDrag(onDrop));
    act(() => result.current.startDrag(event, press(50, 1025, touch)));

    act(() => vi.advanceTimersByTime(299));
    expect(result.current.drag).toBeNull();
    act(() => vi.advanceTimersByTime(1));
    // Lifted in place until the finger moves.
    expect(result.current.drag).toEqual({
      entryId: "event-1",
      start: new Date(2026, 9, 15, 10, 0),
    });
    // The page must not scroll under the finger.
    expect(fireEvent.touchMove(window)).toBe(false);

    pointer("pointerMove", 150, 1440);
    pointer("pointerUp", 150, 1440);
    expect(onDrop).toHaveBeenCalledWith(
      expect.objectContaining({
        starts_at: new Date(2026, 9, 16, 14, 0).toISOString(),
      }),
    );
  });

  it("keeps the grid from scrolling after the pressed block is redrawn", () => {
    const { result } = renderHook(() => useEntryDrag(vi.fn()));
    act(() => result.current.startDrag(event, press(50, 1025, touch)));
    act(() => vi.advanceTimersByTime(300));
    pointer("pointerMove", 150, 1440);

    // Dragged into another column, the block's old element leaves the page,
    // but the finger's touch events keep going to it.
    item.remove();
    expect(fireEvent.touchMove(item)).toBe(false);
  });

  it("leaves a quick tap to open the item", () => {
    const onDrop = vi.fn();
    const { result } = renderHook(() => useEntryDrag(onDrop));
    act(() => result.current.startDrag(event, press(50, 1025, touch)));
    act(() => vi.advanceTimersByTime(100));
    pointer("pointerUp", 50, 1025);

    act(() => vi.advanceTimersByTime(500));
    expect(result.current.drag).toBeNull();
    expect(onDrop).not.toHaveBeenCalled();
  });

  it("leaves a moving finger to scrolling and swiping", () => {
    const { result } = renderHook(() => useEntryDrag(vi.fn()));
    act(() => result.current.startDrag(event, press(50, 1025, touch)));
    act(() => vi.advanceTimersByTime(100));
    pointer("pointerMove", 50, 1060);

    act(() => vi.advanceTimersByTime(500));
    expect(result.current.drag).toBeNull();
    expect(fireEvent.touchMove(window)).toBe(true);
  });

  it("does not save a long press released without moving", () => {
    const onDrop = vi.fn();
    const { result } = renderHook(() => useEntryDrag(onDrop));
    // An event that starts off the half-hour grid stays where it was.
    const offGrid = {
      ...event,
      starts_at: new Date(2026, 9, 15, 10, 10).toISOString(),
    };
    act(() => result.current.startDrag(offGrid, press(50, 1025, touch)));
    act(() => vi.advanceTimersByTime(300));
    pointer("pointerUp", 50, 1025);
    expect(onDrop).not.toHaveBeenCalled();
  });
});

describe("useEntryDrag auto-scroll", () => {
  beforeEach(() => vi.useFakeTimers());
  afterEach(() => vi.useRealTimers());

  // An 800px tall scroller over the 2400px grid; the columns move up as it
  // scrolls.
  const scrollable = () => {
    const scroller = document.querySelector<HTMLElement>(".week-calendar")!;
    let scrollTop = 0;
    Object.defineProperty(scroller, "scrollTop", {
      get: () => scrollTop,
      set: (value: number) => {
        scrollTop = Math.max(0, Math.min(1600, value));
      },
    });
    scroller.getBoundingClientRect = () => new DOMRect(0, 0, 200, 800);
    document
      .querySelectorAll<HTMLElement>(".day-column")
      .forEach((column, i) => {
        column.getBoundingClientRect = () =>
          new DOMRect(i * 100, -scrollTop, 100, 2400);
      });
    return scroller;
  };

  it("scrolls while the item is held near the bottom edge, and stops away from it", () => {
    const scroller = scrollable();
    const { result } = renderHook(() => useEntryDrag(vi.fn()));
    act(() => result.current.startDrag(event, press(50, 400)));

    pointer("pointerMove", 50, 790);
    const startBefore = result.current.drag!.start;
    act(() => vi.advanceTimersByTime(500));
    expect(scroller.scrollTop).toBeGreaterThan(0);
    // The grid moved under the pointer, so the item moved later in the day.
    expect(result.current.drag!.start.getTime()).toBeGreaterThan(
      startBefore.getTime(),
    );

    pointer("pointerMove", 50, 400);
    const scrolled = scroller.scrollTop;
    act(() => vi.advanceTimersByTime(500));
    expect(scroller.scrollTop).toBe(scrolled);
  });

  it("does not scroll before a press becomes a drag", () => {
    const scroller = scrollable();
    const { result } = renderHook(() => useEntryDrag(vi.fn()));
    act(() => result.current.startDrag(event, press(50, 790)));
    pointer("pointerMove", 51, 791);
    act(() => vi.advanceTimersByTime(500));
    expect(scroller.scrollTop).toBe(0);
  });
});
