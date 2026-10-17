import { describe, expect, it } from "vitest";
import {
  getDayCapacity,
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
      "user-1",
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
      "user-1",
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
      "user-1",
    );
    expect(getTransitCardPillVariant(views[3])).toBe("conflict");
  });
});

describe("getTransitCardSubmission", () => {
  it("submits only days that have not ended, including an empty selection", () => {
    const views = getTransitCardDayViews(
      dates,
      [],
      new Date(2026, 9, 22, 9),
      "user-1",
    );
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
    const views = getTransitCardDayViews(
      dates,
      [],
      new Date(2026, 9, 27),
      "user-1",
    );
    expect(getTransitCardSubmission(views, new Map())).toBeNull();
  });
});

describe("household members", () => {
  const members = [
    { user_id: "user-1", display_name: "Jonathan" },
    { user_id: "user-2", display_name: "Anna" },
    { user_id: "user-3", display_name: "Erik" },
  ];
  const memberDay = (
    user_id: string,
    date: string,
    need: TransitCardNeed,
  ): TransitCardDay => ({
    ...reported(date, need),
    user_id,
    display_name: members.find((member) => member.user_id === user_id)!
      .display_name,
  });
  const days = [
    memberDay("user-2", "2026-10-22", "yes"),
    memberDay("user-3", "2026-10-22", "maybe"),
    memberDay("user-1", "2026-10-23", "no"),
  ];

  it("builds the user's own view and lists everyone else separately", () => {
    const views = getTransitCardDayViews(
      dates,
      days,
      new Date(2026, 9, 19),
      "user-1",
      members,
    );

    expect(views[3].state).toBe("not_reported");
    expect(views[3].day).toBeUndefined();
    expect(views[3].others).toEqual([
      { user_id: "user-2", display_name: "Anna", need: "yes" },
      { user_id: "user-3", display_name: "Erik", need: "maybe" },
    ]);
    expect(views[4].state).toBe("not_needed");
    expect(views[4].others.map((member) => member.need)).toEqual([
      undefined,
      undefined,
    ]);
  });

  it("counts the live choice against the card count", () => {
    const [view] = getTransitCardDayViews(
      [dates[3]],
      days,
      new Date(2026, 9, 19),
      "user-1",
      members,
    );

    expect(getDayCapacity(view, "maybe", 1)).toEqual({
      yesCount: 1,
      overCapacity: false,
    });
    expect(getDayCapacity(view, "yes", 1)).toEqual({
      yesCount: 2,
      overCapacity: true,
    });
    expect(getDayCapacity(view, "yes", 2).overCapacity).toBe(false);
  });
});
