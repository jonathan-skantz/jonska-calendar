import { describe, expect, it } from "vitest";
import {
  DEFAULT_CALENDAR_PREFERENCES,
  getEffectiveWeekStart,
  parseCalendarPreferences,
  weekdayName,
} from "./calendarPreferences";

describe("calendar preferences", () => {
  it("uses browser defaults when no saved preferences exist", () => {
    expect(parseCalendarPreferences(null)).toEqual(
      DEFAULT_CALENDAR_PREFERENCES,
    );
    expect(parseCalendarPreferences("invalid")).toEqual(
      DEFAULT_CALENDAR_PREFERENCES,
    );
  });

  it("accepts valid preferences and falls back for invalid fields", () => {
    expect(
      parseCalendarPreferences(
        JSON.stringify({ timeFormat: "24-hour", weekStartsOn: 6 }),
      ),
    ).toEqual({ timeFormat: "24-hour", weekStartsOn: 6 });
    expect(
      parseCalendarPreferences(
        JSON.stringify({ timeFormat: "invalid", weekStartsOn: 9 }),
      ),
    ).toEqual(DEFAULT_CALENDAR_PREFERENCES);
  });

  it("resolves browser week settings and localizes weekday names", () => {
    expect(getEffectiveWeekStart(2)).toBe(2);
    expect(weekdayName(0, "en-US")).toBe("Sunday");
  });
});
