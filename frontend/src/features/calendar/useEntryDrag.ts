import { useEffect, useEffectEvent, useRef, useState } from "react";
import type { TimelineEntry } from "../../types";
import {
  canDrag,
  getDropStart,
  getGrabOffset,
  moveEntry,
  timeFromPoint,
} from "./dragMove";
import type { DragColumn } from "./dragMove";

// How far a mouse or pen moves before a press becomes a drag; less is a click.
const DRAG_THRESHOLD_PX = 4;
// A finger picks an item up by holding still this long. Moving further than
// the slop first leaves the touch to scrolling and swiping.
const LONG_PRESS_MS = 300;
const LONG_PRESS_SLOP_PX = 8;

// The item being dragged and where it would land. After the drop it stays
// there, no longer lifted, until its save has settled.
export type EntryDrag = { entryId: string; start: Date; dropped?: boolean };

type DragPointer = Pick<
  PointerEvent,
  "button" | "isPrimary" | "pointerType" | "clientX" | "clientY"
> & { currentTarget: EventTarget | null; target: EventTarget | null };

type Press = {
  entry: TimelineEntry;
  scroller: Element;
  x: number;
  y: number;
  grabOffset: number;
  touch: boolean;
  // Where the finger went down; its touch events keep going there.
  target: EventTarget | null;
};

// The visible day columns, measured now: they move as the grid scrolls.
const readColumns = (scroller: Element): DragColumn[] =>
  Array.from(
    scroller.querySelectorAll<HTMLElement>(".day-column[data-date]"),
  ).map((column) => ({
    date: new Date(`${column.dataset.date}T00:00`),
    rect: column.getBoundingClientRect(),
  }));

// A click follows the pointerup that ends a drag; it must not open the item
// or start a new one in the day column underneath.
const swallowNextClick = () => {
  const swallow = (event: MouseEvent) => {
    event.preventDefault();
    event.stopPropagation();
  };
  window.addEventListener("click", swallow, { capture: true, once: true });
  setTimeout(() =>
    window.removeEventListener("click", swallow, { capture: true }),
  );
};

// Moves events and tasks in the week view. A mouse or pen press becomes a
// drag once the pointer moves; a finger has to hold still first. `drag` then
// holds the snapped start the item would get, and releasing calls onDrop with
// the moved item. While the promise onDrop returns is pending, `drag` keeps
// the item at its new place, so it never flashes back to where it was.
export const useEntryDrag = (
  onDrop: (moved: TimelineEntry) => Promise<unknown> | void,
) => {
  const [press, setPress] = useState<Press | null>(null);
  const [drag, setDrag] = useState<EntryDrag | null>(null);
  const drop = useEffectEvent(onDrop);
  // Set while a finger drags an item, so the grid must not scroll under it.
  const holdingScroll = useRef(false);

  // A browser only lets a touch be kept from scrolling if a non-passive
  // touchmove listener existed when the touch began, so this one is always
  // there; it does nothing unless a finger is dragging.
  useEffect(() => {
    const holdScroll = (event: TouchEvent) => {
      if (holdingScroll.current) event.preventDefault();
    };
    window.addEventListener("touchmove", holdScroll, { passive: false });
    return () => window.removeEventListener("touchmove", holdScroll);
  }, []);

  useEffect(() => {
    if (!press) return;
    // Picked up: the press is a drag, and the item follows the pointer.
    let picked = false;
    let start: Date | null = null;
    const startTime = new Date(
      (press.entry.type === "task"
        ? press.entry.deadline_at
        : press.entry.starts_at) ?? 0,
    );

    const pickUp = () => {
      picked = true;
      holdingScroll.current = press.touch;
      document.body.classList.add("dragging-entry");
    };
    const finish = () => {
      holdingScroll.current = false;
      document.body.classList.remove("dragging-entry");
      setDrag(null);
      setPress(null);
    };
    // Lifts a held item in place, before the finger moves it anywhere.
    const longPress = press.touch
      ? setTimeout(() => {
          pickUp();
          navigator.vibrate?.(10);
          setDrag({ entryId: press.entry.id, start: startTime });
        }, LONG_PRESS_MS)
      : undefined;
    const move = (event: PointerEvent) => {
      const distance = Math.hypot(
        event.clientX - press.x,
        event.clientY - press.y,
      );
      if (!picked) {
        if (press.touch) {
          if (distance > LONG_PRESS_SLOP_PX) finish();
          return;
        }
        if (distance < DRAG_THRESHOLD_PX) return;
        pickUp();
      }
      const time = timeFromPoint(
        readColumns(press.scroller),
        event.clientX,
        event.clientY,
      );
      if (!time) return;
      const next = getDropStart(time, press.grabOffset);
      if (start?.getTime() !== next.getTime()) {
        start = next;
        setDrag({ entryId: press.entry.id, start: next });
      }
    };
    const release = (event: PointerEvent) => {
      if (picked) {
        // Tells the week view's swipe handler this was not a swipe.
        event.preventDefault();
        swallowNextClick();
        if (start && start.getTime() !== startTime.getTime()) {
          const landed = { entryId: press.entry.id, start, dropped: true };
          finish();
          setDrag(landed);
          // A failure is shown by the caller; the item then goes back.
          Promise.resolve(drop(moveEntry(press.entry, start)))
            .catch(() => {})
            .finally(() =>
              setDrag((current) => (current === landed ? null : current)),
            );
          return;
        }
      }
      finish();
    };
    const cancelOnEscape = (event: KeyboardEvent) => {
      if (event.key === "Escape") finish();
    };
    // The dragged block is redrawn as it moves and can leave the page, e.g.
    // into another day's column. Its touch events still go to the element
    // the finger went down on, but no longer reach the window from there.
    const holdScroll = (event: Event) => {
      if (picked) event.preventDefault();
    };
    // A long press would otherwise open the browser's context menu.
    const blockContextMenu = (event: Event) => event.preventDefault();

    // Capture, so the drag sees the pointer before anything under it.
    window.addEventListener("pointermove", move, true);
    window.addEventListener("pointerup", release, true);
    window.addEventListener("pointercancel", finish, true);
    window.addEventListener("keydown", cancelOnEscape);
    if (press.touch) {
      press.target?.addEventListener("touchmove", holdScroll, {
        passive: false,
      });
      window.addEventListener("contextmenu", blockContextMenu, true);
    }
    return () => {
      clearTimeout(longPress);
      window.removeEventListener("pointermove", move, true);
      window.removeEventListener("pointerup", release, true);
      window.removeEventListener("pointercancel", finish, true);
      window.removeEventListener("keydown", cancelOnEscape);
      press.target?.removeEventListener("touchmove", holdScroll);
      window.removeEventListener("contextmenu", blockContextMenu, true);
      holdingScroll.current = false;
      document.body.classList.remove("dragging-entry");
    };
  }, [press]);

  const startDrag = (entry: TimelineEntry, event: DragPointer) => {
    if (!canDrag(entry) || !event.isPrimary || event.button !== 0) return;
    const scroller =
      event.currentTarget instanceof Element
        ? event.currentTarget.closest(".week-calendar")
        : null;
    const time =
      scroller &&
      timeFromPoint(readColumns(scroller), event.clientX, event.clientY);
    if (!scroller || !time) return;
    setPress({
      entry,
      scroller,
      x: event.clientX,
      y: event.clientY,
      grabOffset: getGrabOffset(entry, time),
      touch: event.pointerType === "touch",
      target: event.target,
    });
  };

  return { drag, startDrag };
};
