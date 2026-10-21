import { describe, expect, it } from "vitest";
import {
  describeRecurrence,
  monthlyOptions,
  readRecurrence,
  weekdayOf,
} from "./recurrence";

// Tuesday 13 October 2026: the 2nd Tuesday of its month.
const tuesday = new Date(2026, 9, 13, 10);
// Friday 30 October 2026: the last (and 5th) Friday of its month.
const lastFriday = new Date(2026, 9, 30, 18);

const form = (entries: [string, string][]) => {
  const data = new FormData();
  for (const [name, value] of entries) data.append(name, value);
  return data;
};

describe("recurrence helpers", () => {
  it("names the weekday of a date", () => {
    expect(weekdayOf(tuesday)).toBe("TU");
    expect(weekdayOf(new Date(2026, 9, 18))).toBe("SU");
  });

  it("offers the monthly days a start allows", () => {
    expect(monthlyOptions(tuesday).map(({ label }) => label)).toEqual([
      "On day 13",
      "On the 2nd Tuesday",
    ]);
    expect(monthlyOptions(lastFriday).map(({ value }) => value)).toEqual([
      "day_of_month",
      "last_weekday",
    ]);
  });

  it("describes rules", () => {
    expect(
      describeRecurrence({ frequency: "daily", interval: 1 }, tuesday),
    ).toBe("Daily");
    expect(
      describeRecurrence(
        { frequency: "weekly", interval: 2, weekdays: ["TU", "TH"], count: 10 },
        tuesday,
      ),
    ).toBe("Every 2 weeks on Tue, Thu, 10 times");
    expect(
      describeRecurrence(
        {
          frequency: "monthly",
          interval: 1,
          monthly_by: "weekday",
          until: "2026-12-31",
        },
        tuesday,
      ),
    ).toBe("Monthly on the 2nd Tuesday, until Dec 31, 2026");
    expect(
      describeRecurrence(
        { frequency: "yearly", interval: 1, count: 1 },
        tuesday,
      ),
    ).toBe("Yearly, 1 time");
  });

  it("reads the rule from the form", () => {
    expect(readRecurrence(form([["repeat_frequency", "none"]]))).toBeNull();
    expect(
      readRecurrence(
        form([
          ["repeat_frequency", "weekly"],
          ["repeat_interval", "2"],
          ["repeat_weekday", "TH"],
          ["repeat_weekday", "TU"],
          ["repeat_ends", "after"],
          ["repeat_count", "10"],
        ]),
      ),
    ).toEqual({
      frequency: "weekly",
      interval: 2,
      weekdays: ["TU", "TH"],
      count: 10,
    });
    expect(
      readRecurrence(
        form([
          ["repeat_frequency", "monthly"],
          ["repeat_interval", "1"],
          ["repeat_monthly_by", "last_weekday"],
          ["repeat_ends", "on"],
          ["repeat_until", "2027-06-30"],
        ]),
      ),
    ).toEqual({
      frequency: "monthly",
      interval: 1,
      monthly_by: "last_weekday",
      until: "2027-06-30",
    });
  });
});
