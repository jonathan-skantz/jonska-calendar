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

  const gatheringEntry: TimelineEntry = {
    ...entry,
    kind: "attendable",
    starts_at: "2026-10-15T18:00:00",
    ends_at: "2026-10-15T20:00:00",
    gathering_seconds: 30 * 60,
  };
  const renderGathering = (
    item: TimelineEntry,
    isOptional: boolean,
    startMinutes: number,
  ) =>
    render(
      <CalendarEntry
        entry={item}
        startMinutes={startMinutes}
        durationMinutes={1200 - startMinutes}
        trimTop={false}
        trimBottom={false}
        overlapDepth={0}
        continuesFromPreviousDay={false}
        continuesToNextDay={false}
        commuteSegments={[]}
        gatheringSegment={{
          minutes: 30,
          isOptional,
          label: item.gathering_label,
          startMinutes: 1050,
          durationMinutes: 30,
          endsAtEventStart: true,
        }}
        showBody
        timeFormat="24-hour"
        onEdit={vi.fn()}
        onToggleTask={vi.fn()}
      />,
    );

  it("draws mandatory gathering as part of the block with a start line", () => {
    const { container } = renderGathering(
      { ...gatheringEntry, gathering_label: "Warm-up" },
      false,
      1050,
    );

    // The approximate start moves to when people gather.
    expect(container.querySelector(".gathering-label")?.textContent).toBe(
      "Warm-up ~17:30",
    );
    expect(container.querySelector("article .gathering-start")).toBeTruthy();
    expect(container.querySelector(".gathering-band")).toBeNull();
    expect(
      container
        .querySelector("article")
        ?.classList.contains("approximate-start"),
    ).toBe(true);
    expect(screen.getByText("18:00 – 20:00")).toBeTruthy();
  });

  it("draws optional gathering as a separate dotted band", () => {
    const { container } = renderGathering(
      { ...gatheringEntry, gathering_is_optional: true },
      true,
      1080,
    );

    const band = screen.getByRole("img", {
      name: "Optional gathering from ~17:30",
    });
    expect(band.classList.contains("optional")).toBe(true);
    expect(band.classList.contains("approximate-start")).toBe(true);
    expect(band.textContent).toBe("Gather ~17:30");
    const article = container.querySelector("article");
    expect(article?.querySelector(".gathering-start")).toBeNull();
    expect(article?.classList.contains("has-optional-gathering")).toBe(true);
    expect(article?.classList.contains("approximate-start")).toBe(false);
  });
});
