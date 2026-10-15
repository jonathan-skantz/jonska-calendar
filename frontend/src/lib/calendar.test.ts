import { describe, expect, it } from "vitest";
import { formatHour, getWeekStartsOn, startOfCalendarWeek } from "./calendar";

describe("browser locale calendar preferences", () => {
  it("uses the locale's first day of the week", () => {
    expect(getWeekStartsOn("en-US")).toBe(0);
    expect(getWeekStartsOn("sv-SE")).toBe(1);
  });

  it("starts weeks on the locale's first weekday", () => {
    const date = new Date(2026, 9, 15);

    expect(startOfCalendarWeek(date).getDay()).toBe(
      getWeekStartsOn(Intl.DateTimeFormat().resolvedOptions().locale),
    );
  });

  it("uses the locale's default hour cycle for the hour labels", () => {
    expect(formatHour(13, { locale: "en-US" })).toMatch(/1:00\s*PM/i);
    expect(formatHour(13, { locale: "sv-SE" })).toBe("13:00");
  });

  it("allows an explicit 12-hour or 24-hour format", () => {
    expect(formatHour(13, { locale: "sv-SE", preference: "12-hour" })).toMatch(
      /1:00\s*em/i,
    );
    expect(formatHour(13, { locale: "en-US", preference: "24-hour" })).toMatch(
      /^13:00$/,
    );
  });
});
