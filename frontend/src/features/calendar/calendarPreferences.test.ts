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
    ).toEqual({
      ...DEFAULT_CALENDAR_PREFERENCES,
      timeFormat: "24-hour",
      weekStartsOn: 6,
    });
    expect(
      parseCalendarPreferences(
        JSON.stringify({ timeFormat: "invalid", weekStartsOn: 9 }),
      ),
    ).toEqual(DEFAULT_CALENDAR_PREFERENCES);
  });

  it("keeps saved sun toggles and defaults them to on", () => {
    expect(
      parseCalendarPreferences(
        JSON.stringify({
          showSunrise: false,
          showSunset: "no",
          showNight: false,
        }),
      ),
    ).toEqual({
      ...DEFAULT_CALENDAR_PREFERENCES,
      showSunrise: false,
      showSunset: true,
      showNight: false,
    });
  });

  it("keeps a saved zoom level and falls back to the full day", () => {
    expect(
      parseCalendarPreferences(JSON.stringify({ visibleHours: 12 })),
    ).toEqual({ ...DEFAULT_CALENDAR_PREFERENCES, visibleHours: 12 });
    expect(
      parseCalendarPreferences(JSON.stringify({ visibleHours: 7 }))
        .visibleHours,
    ).toBe(24);
  });

  it("resolves browser week settings and localizes weekday names", () => {
    expect(getEffectiveWeekStart(2)).toBe(2);
    expect(weekdayName(0, "en-US")).toBe("Sunday");
  });
});
