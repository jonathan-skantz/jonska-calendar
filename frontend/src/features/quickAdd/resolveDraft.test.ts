import { describe, expect, it } from "vitest";
import { dateKey } from "../../lib/calendar";
import { resolveDate, resolveDraft } from "./resolveDraft";

// Wednesday 14 October 2026, mid-afternoon local time.
const today = new Date(2026, 9, 14, 15, 30);
const MONDAY = 1;
const SUNDAY = 0;

const day = (date: Date | null) => (date ? dateKey(date) : null);
const local = (iso?: string) => (iso ? new Date(iso) : null);

describe("resolveDate", () => {
  it("counts relative days from today", () => {
    expect(
      day(resolveDate({ kind: "relative_days", days: 0 }, today, MONDAY)),
    ).toBe("2026-10-14");
    expect(
      day(resolveDate({ kind: "relative_days", days: 2 }, today, MONDAY)),
    ).toBe("2026-10-16");
  });

  it("takes the next occurrence of a weekday, today included", () => {
    expect(
      day(resolveDate({ kind: "weekday", weekday: "friday" }, today, MONDAY)),
    ).toBe("2026-10-16");
    expect(
      day(
        resolveDate({ kind: "weekday", weekday: "wednesday" }, today, MONDAY),
      ),
    ).toBe("2026-10-14");
    expect(
      day(resolveDate({ kind: "weekday", weekday: "monday" }, today, MONDAY)),
    ).toBe("2026-10-19");
  });

  it("takes next week's weekday by the user's week start", () => {
    const nextFriday = {
      kind: "weekday",
      weekday: "friday",
      week_offset: 1,
    } as const;
    expect(day(resolveDate(nextFriday, today, MONDAY))).toBe("2026-10-23");
    expect(day(resolveDate(nextFriday, today, SUNDAY))).toBe("2026-10-23");
    // Sunday ends a Monday week but starts a Sunday week.
    const nextSunday = {
      kind: "weekday",
      weekday: "sunday",
      week_offset: 1,
    } as const;
    expect(day(resolveDate(nextSunday, today, MONDAY))).toBe("2026-10-25");
    expect(day(resolveDate(nextSunday, today, SUNDAY))).toBe("2026-10-18");
  });

  it("rolls a calendar date without a year into next year once it has passed", () => {
    expect(
      day(resolveDate({ kind: "calendar", month: 10, day: 24 }, today, MONDAY)),
    ).toBe("2026-10-24");
    expect(
      day(resolveDate({ kind: "calendar", month: 10, day: 14 }, today, MONDAY)),
    ).toBe("2026-10-14");
    expect(
      day(resolveDate({ kind: "calendar", month: 3, day: 1 }, today, MONDAY)),
    ).toBe("2027-03-01");
    expect(
      day(
        resolveDate(
          { kind: "calendar", month: 3, day: 1, year: 2026 },
          today,
          MONDAY,
        ),
      ),
    ).toBe("2026-03-01");
  });

  it("rejects dates that do not exist", () => {
    expect(
      resolveDate({ kind: "calendar", month: 2, day: 30 }, today, MONDAY),
    ).toBeNull();
  });
});

describe("resolveDraft", () => {
  it("builds a timed event from stated fields", () => {
    const resolved = resolveDraft(
      {
        title: "Middag",
        date: { kind: "relative_days", days: 1 },
        start_time: "18:00",
        end_time: "21:00",
        start_is_approximate: true,
        location: "mormor",
        commute_before_minutes: 30,
        tag_ids: ["family"],
        reminder_minutes: [60, 60],
      },
      today,
      MONDAY,
      "2026-10-14",
    );
    expect(resolved.values).toMatchObject({
      title: "Middag",
      is_all_day: false,
      start_is_approximate: true,
      location: "mormor",
      commute_before_seconds: 1800,
    });
    expect(local(resolved.values.starts_at)).toEqual(
      new Date(2026, 9, 15, 18, 0),
    );
    expect(local(resolved.values.ends_at)).toEqual(
      new Date(2026, 9, 15, 21, 0),
    );
    expect(resolved.tagIds).toEqual(["family"]);
    expect(resolved.reminderOffsets).toEqual([3600]);
    expect(resolved.warnings).toEqual([]);
  });

  it("uses a duration, then the default length", () => {
    const withDuration = resolveDraft(
      {
        date: { kind: "relative_days", days: 1 },
        start_time: "10:00",
        duration_minutes: 45,
      },
      today,
      MONDAY,
      "2026-10-14",
    );
    expect(local(withDuration.values.ends_at)).toEqual(
      new Date(2026, 9, 15, 10, 45),
    );

    const withDefault = resolveDraft(
      { date: { kind: "relative_days", days: 1 }, start_time: "10:00" },
      today,
      MONDAY,
      "2026-10-14",
    );
    expect(local(withDefault.values.ends_at)).toEqual(
      new Date(2026, 9, 15, 12, 0),
    );
  });

  it("moves an end before the start to the next day", () => {
    const resolved = resolveDraft(
      {
        date: { kind: "relative_days", days: 1 },
        start_time: "22:00",
        end_time: "02:00",
      },
      today,
      MONDAY,
      "2026-10-14",
    );
    expect(local(resolved.values.ends_at)).toEqual(new Date(2026, 9, 16, 2, 0));
    expect(resolved.warnings).toHaveLength(1);
  });

  it("ends all-day events at midnight after the last day", () => {
    const single = resolveDraft(
      {
        date: { kind: "calendar", month: 10, day: 24 },
        all_day: true,
        start_is_approximate: true,
      },
      today,
      MONDAY,
      "2026-10-14",
    );
    expect(local(single.values.starts_at)).toEqual(new Date(2026, 9, 24));
    expect(local(single.values.ends_at)).toEqual(new Date(2026, 9, 25));
    expect(single.values.start_is_approximate).toBe(false);
    expect(single.warnings).toEqual([]);

    const multi = resolveDraft(
      {
        date: { kind: "calendar", month: 10, day: 24 },
        end_date: { kind: "calendar", month: 10, day: 26 },
        all_day: true,
      },
      today,
      MONDAY,
      "2026-10-14",
    );
    expect(local(multi.values.ends_at)).toEqual(new Date(2026, 9, 27));
  });

  it("falls back to the selected day at 09:00 and says so", () => {
    const resolved = resolveDraft(
      { title: "Gym" },
      today,
      MONDAY,
      "2026-10-20",
    );
    expect(local(resolved.values.starts_at)).toEqual(
      new Date(2026, 9, 20, 9, 0),
    );
    expect(resolved.warnings).toHaveLength(2);
  });

  it("warns about past and impossible dates", () => {
    expect(
      resolveDraft(
        {
          date: { kind: "calendar", month: 10, day: 1, year: 2026 },
          start_time: "10:00",
        },
        today,
        MONDAY,
        "2026-10-14",
      ).warnings,
    ).toEqual(["That date has passed"]);
    expect(
      resolveDraft(
        { date: { kind: "calendar", month: 2, day: 30 }, start_time: "10:00" },
        today,
        MONDAY,
        "2026-10-14",
      ).warnings,
    ).toEqual(["That date doesn't exist"]);
  });
});
