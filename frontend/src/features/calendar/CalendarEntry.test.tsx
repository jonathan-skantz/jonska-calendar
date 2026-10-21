import { cleanup, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { CalendarEntry } from "./CalendarEntry";
import type { TimelineEntry } from "../../types";
import { renderWithPreferences } from "../../test/renderWithPreferences";

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
    const { container } = renderWithPreferences(
      <CalendarEntry
        entry={entry}
        startMinutes={600}
        durationMinutes={480}
        trimTop={false}
        trimBottom={false}
        placement={{ left: 0, right: 1, raised: false }}
        continuesFromPreviousDay={false}
        continuesToNextDay={false}
        commuteSegments={[]}
        showBody
        onEdit={vi.fn()}
        onToggleTask={vi.fn()}
      />,
    );

    const article = container.querySelector("article");
    expect(article?.classList.contains("approximate-start")).toBe(true);
    expect(article?.classList.contains("approximate-end")).toBe(false);
    expect(screen.getByText("~10:00 – 18:00").title).toBe("Approximate start");
  });

  it("keeps the full text in a tooltip for short or narrow blocks", () => {
    const { container } = renderWithPreferences(
      <CalendarEntry
        entry={{ ...entry, location: "Lab", shared_by: "Ana" }}
        startMinutes={600}
        durationMinutes={480}
        trimTop={false}
        trimBottom={false}
        placement={{ left: 0, right: 1, raised: false }}
        continuesFromPreviousDay={false}
        continuesToNextDay={false}
        commuteSegments={[]}
        showBody
        onEdit={vi.fn()}
        onToggleTask={vi.fn()}
      />,
    );

    expect(container.querySelector("article")?.title).toBe(
      "Workshop\n~10:00 – 18:00\nLab\nfrom Ana",
    );
    // Shown instead of the detail lines when the block is one line tall.
    expect(container.querySelector(".entry-start")?.textContent).toBe(
      "~10:00 ",
    );
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
    renderWithPreferences(
      <CalendarEntry
        entry={item}
        startMinutes={startMinutes}
        durationMinutes={1200 - startMinutes}
        trimTop={false}
        trimBottom={false}
        placement={{ left: 0, right: 1, raised: false }}
        continuesFromPreviousDay={false}
        continuesToNextDay={false}
        commuteSegments={[]}
        gatheringSegment={{
          minutes: 30,
          isOptional,
          label: item.gathering_label,
          startMinutes: 1050,
          durationMinutes: 30,
          continuesFromPreviousDay: false,
          endsAtEventStart: true,
        }}
        showBody
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

  const renderEntry = (item: TimelineEntry) =>
    renderWithPreferences(
      <CalendarEntry
        entry={item}
        startMinutes={600}
        durationMinutes={120}
        trimTop={false}
        trimBottom={false}
        placement={{ left: 0, right: 1, raised: false }}
        continuesFromPreviousDay={false}
        continuesToNextDay={false}
        commuteSegments={[]}
        showBody
        onEdit={vi.fn()}
        onToggleTask={vi.fn()}
      />,
    );
  const lecture: TimelineEntry = {
    ...entry,
    title: "Lecture",
    start_is_approximate: false,
    starts_at: "2026-10-15T10:00:00",
    ends_at: "2026-10-15T12:00:00",
  };

  it("draws heads-up events as an outlined ghost", () => {
    const { container } = renderEntry({ ...lecture, kind: "informational" });
    const article = container.querySelector("article")!;
    expect(article.className).toContain("not-busy heads-up");
    expect(article.style.backgroundColor).toBe("");
    expect(screen.getByRole("img", { name: "Heads-up" })).toBeTruthy();
  });

  it("marks skipped events and hides their commute", () => {
    const { container } = renderEntry({
      ...lecture,
      kind: "attendable",
      is_skipped: true,
      commute_before_seconds: 1200,
    });
    expect(container.querySelector("article")!.className).toContain(
      "not-busy skipped",
    );
    expect(screen.getByRole("img", { name: "Skipping" })).toBeTruthy();
    expect(screen.getByText("Skipping · 10:00 – 12:00")).toBeTruthy();
    expect(container.querySelector(".commute-band")).toBeNull();
  });

  it("keeps attended events solid", () => {
    const { container } = renderEntry({ ...lecture, kind: "attendable" });
    const article = container.querySelector("article")!;
    expect(article.className).not.toContain("not-busy");
    expect(article.style.backgroundColor).not.toBe("");
  });

  it("marks events that have reminders", () => {
    renderWithPreferences(
      <CalendarEntry
        entry={{ ...lecture, kind: "attendable" }}
        startMinutes={600}
        durationMinutes={120}
        trimTop={false}
        trimBottom={false}
        placement={{ left: 0, right: 1, raised: false }}
        continuesFromPreviousDay={false}
        continuesToNextDay={false}
        commuteSegments={[]}
        showBody
        onEdit={vi.fn()}
        onToggleTask={vi.fn()}
        reminders={[
          {
            id: "reminder-1",
            event_id: "event-1",
            offset_seconds: 900,
            remind_at: "2026-10-15T09:45:00",
            event_title: "Lecture",
            event_starts_at: "2026-10-15T10:00:00",
            occurrence_start: "2026-10-15T10:00:00",
            event_ends_at: "2026-10-15T12:00:00",
          },
        ]}
      />,
    );
    expect(
      screen.getByRole("img", { name: "Reminders: 15 min before" }),
    ).toBeTruthy();
  });

  it("fades approximate edges only where the event really starts or ends", () => {
    // Continuation markers size themselves with a ResizeObserver.
    vi.stubGlobal(
      "ResizeObserver",
      class {
        observe() {}
        disconnect() {}
      },
    );
    const nightShift: TimelineEntry = {
      ...lecture,
      kind: "attendable",
      start_is_approximate: true,
      end_is_approximate: true,
      starts_at: "2026-10-15T22:00:00",
      ends_at: "2026-10-16T02:00:00",
    };
    const renderSegment = (fromPrevious: boolean, toNext: boolean) =>
      renderWithPreferences(
        <CalendarEntry
          entry={nightShift}
          startMinutes={fromPrevious ? 0 : 1320}
          durationMinutes={120}
          trimTop={false}
          trimBottom={false}
          placement={{ left: 0, right: 1, raised: false }}
          continuesFromPreviousDay={fromPrevious}
          continuesToNextDay={toNext}
          commuteSegments={[]}
          showBody
          onEdit={vi.fn()}
          onToggleTask={vi.fn()}
        />,
      ).container.querySelector("article")!.className;

    const firstDay = renderSegment(false, true);
    expect(firstDay).toContain("approximate-start");
    expect(firstDay).not.toContain("approximate-end");
    cleanup();
    const secondDay = renderSegment(true, false);
    expect(secondDay).not.toContain("approximate-start");
    expect(secondDay).toContain("approximate-end");
    vi.unstubAllGlobals();
  });

  it("keeps a short event's title with mandatory gathering", () => {
    const { container } = renderGathering(
      {
        ...gatheringEntry,
        title: "Quick sync",
        starts_at: "2026-10-15T18:00:00",
        ends_at: "2026-10-15T18:15:00",
        start_is_approximate: false,
      },
      false,
      1050,
    );
    const article = container.querySelector("article")!;
    // Content is no longer pushed below the start line.
    expect(article.style.paddingTop).toBe("");
    expect(article.firstElementChild?.className).toBe("gathering-label");
    expect(article.querySelector(".gathering-start")?.tagName).toBe("I");
  });
});
