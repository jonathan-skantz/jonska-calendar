import { describe, expect, it } from "vitest";
import {
  getTransitCardDayViews,
  getTransitCardPillVariant,
  getTransitCardSubmission,
} from "./transitCardDays";
import type { TransitCardDay, TransitCardNeed } from "../../types";

const dates = [19, 20, 21, 22, 23, 24, 25].map((day) => new Date(2026, 9, day));
const reported = (date: string, need: TransitCardNeed): TransitCardDay => ({
  date,
  user_id: "user-1",
  display_name: "Jonathan",
  need,
  status: "resolved",
});

describe("getTransitCardDayViews", () => {
  it("distinguishes past, missing, not needed, maybe and needed days", () => {
    const views = getTransitCardDayViews(
      dates,
      [
        reported("2026-10-19", "yes"),
        reported("2026-10-21", "no"),
        reported("2026-10-22", "yes"),
        reported("2026-10-23", "maybe"),
      ],
      new Date(2026, 9, 21, 23, 30),
    );

    expect(views.map((view) => view.state)).toEqual([
      "past",
      "past",
      "not_needed",
      "resolved",
      "maybe",
      "not_reported",
      "not_reported",
    ]);
    expect(views[0].day?.need).toBe("yes");
  });
});

describe("getTransitCardPillVariant", () => {
  it("shows a pill for every answer and for upcoming unreported days", () => {
    const views = getTransitCardDayViews(
      dates,
      [
        reported("2026-10-19", "yes"),
        reported("2026-10-21", "no"),
        reported("2026-10-22", "yes"),
        reported("2026-10-23", "maybe"),
      ],
      new Date(2026, 9, 21, 9),
    );

    expect(views.map(getTransitCardPillVariant)).toEqual([
      "resolved",
      null,
      "not_needed",
      "resolved",
      "maybe",
      "pending",
      "pending",
    ]);
  });

  it("shows conflicts", () => {
    const views = getTransitCardDayViews(
      dates,
      [{ ...reported("2026-10-22", "yes"), status: "conflict" }],
      new Date(2026, 9, 19),
    );
    expect(getTransitCardPillVariant(views[3])).toBe("conflict");
  });
});

describe("getTransitCardSubmission", () => {
  it("submits only days that have not ended, including an empty selection", () => {
    const views = getTransitCardDayViews(dates, [], new Date(2026, 9, 22, 9));
    expect(getTransitCardSubmission(views, new Map())).toEqual({
      from: "2026-10-22",
      to: "2026-10-25",
      needed_dates: [],
      maybe_dates: [],
    });
    expect(
      getTransitCardSubmission(
        views,
        new Map<string, TransitCardNeed>([
          ["2026-10-20", "yes"],
          ["2026-10-23", "maybe"],
          ["2026-10-24", "yes"],
          ["2026-10-25", "no"],
        ]),
      ),
    ).toEqual({
      from: "2026-10-22",
      to: "2026-10-25",
      needed_dates: ["2026-10-24"],
      maybe_dates: ["2026-10-23"],
    });
  });

  it("returns null when every day has ended", () => {
    const views = getTransitCardDayViews(dates, [], new Date(2026, 9, 27));
    expect(getTransitCardSubmission(views, new Map())).toBeNull();
  });
});
