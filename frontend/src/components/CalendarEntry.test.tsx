import { cleanup, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { CalendarEntry } from "./CalendarEntry";
import type { TimelineEntry } from "../types";

const entry: TimelineEntry = {
  id: "event-1",
  type: "event",
  title: "Workshop",
  starts_at: "2026-10-15T10:00:00",
  ends_at: "2026-10-15T18:00:00",
  start_is_approximate: true,
  is_completed: false,
  tags: [{ id: "work", name: "Work", color: "#336699", is_default: true }],
  main_tag_id: "work",
};

describe("CalendarEntry", () => {
  afterEach(cleanup);

  it("fades approximate edges and marks their times", () => {
    const { container } = render(
      <CalendarEntry
        entry={entry}
        startMinutes={600}
        durationMinutes={480}
        trimTop={false}
        trimBottom={false}
        overlapDepth={0}
        continuesFromPreviousDay={false}
        continuesToNextDay={false}
        commuteSegments={[]}
        showBody
        timeFormat="24-hour"
        onEdit={vi.fn()}
        onToggleTask={vi.fn()}
      />,
    );

    const article = container.querySelector("article");
    expect(article?.classList.contains("approximate-start")).toBe(true);
    expect(article?.classList.contains("approximate-end")).toBe(false);
    expect(screen.getByText("~10:00 – 18:00").title).toBe("Approximate start");
  });
});
