import { useEffect, useEffectEvent, useState } from "react";
import type { TimelineEntry } from "../../types";
import {
  canDrag,
  getDropStart,
  getGrabOffset,
  moveEntry,
  timeFromPoint,
} from "./dragMove";
import type { DragColumn } from "./dragMove";

// How far the pointer moves before a press becomes a drag; less is a click.
const DRAG_THRESHOLD_PX = 4;

// The item being dragged and where it would land.
export type EntryDrag = { entryId: string; start: Date };

type DragPointer = Pick<
  PointerEvent,
  "button" | "isPrimary" | "pointerType" | "clientX" | "clientY"
> & { currentTarget: EventTarget | null };

type Press = {
  entry: TimelineEntry;
  scroller: Element;
  x: number;
  y: number;
  grabOffset: number;
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

// Moves events and tasks in the week view with a mouse or pen. A press on an
// item becomes a drag once the pointer moves; `drag` then holds the snapped
// start the item would get, and releasing calls onDrop with the moved item.
export const useEntryDrag = (onDrop: (moved: TimelineEntry) => void) => {
  const [press, setPress] = useState<Press | null>(null);
  const [drag, setDrag] = useState<EntryDrag | null>(null);
  const drop = useEffectEvent(onDrop);

  useEffect(() => {
    if (!press) return;
    let start: Date | null = null;
    const startTime =
      press.entry.type === "task"
        ? press.entry.deadline_at
        : press.entry.starts_at;

    const finish = () => {
      document.body.classList.remove("dragging-entry");
      setDrag(null);
      setPress(null);
    };
    const move = (event: PointerEvent) => {
      if (
        !start &&
        Math.hypot(event.clientX - press.x, event.clientY - press.y) <
          DRAG_THRESHOLD_PX
      )
        return;
      const time = timeFromPoint(
        readColumns(press.scroller),
        event.clientX,
        event.clientY,
      );
      if (!time) return;
      const next = getDropStart(time, press.grabOffset);
      if (!start) document.body.classList.add("dragging-entry");
      if (start?.getTime() !== next.getTime()) {
        start = next;
        setDrag({ entryId: press.entry.id, start: next });
      }
    };
    const release = (event: PointerEvent) => {
      if (start) {
        // Tells the week view's swipe handler this was not a swipe.
        event.preventDefault();
        swallowNextClick();
        if (start.getTime() !== new Date(startTime ?? 0).getTime()) {
          drop(moveEntry(press.entry, start));
        }
      }
      finish();
    };
    const cancelOnEscape = (event: KeyboardEvent) => {
      if (event.key === "Escape") finish();
    };

    // Capture, so the drag sees the pointer before anything under it.
    window.addEventListener("pointermove", move, true);
    window.addEventListener("pointerup", release, true);
    window.addEventListener("pointercancel", finish, true);
    window.addEventListener("keydown", cancelOnEscape);
    return () => {
      window.removeEventListener("pointermove", move, true);
      window.removeEventListener("pointerup", release, true);
      window.removeEventListener("pointercancel", finish, true);
      window.removeEventListener("keydown", cancelOnEscape);
      document.body.classList.remove("dragging-entry");
    };
  }, [press]);

  const startDrag = (entry: TimelineEntry, event: DragPointer) => {
    if (
      !canDrag(entry) ||
      !event.isPrimary ||
      event.button !== 0 ||
      event.pointerType === "touch"
    )
      return;
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
    });
  };

  return { drag, startDrag };
};
