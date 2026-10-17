import { describe, expect, it } from "vitest";
import { getSunIndicators } from "./sunTimes";
import type { WeatherDay } from "../../types";

const date = new Date(2026, 9, 17);
const allOn = { showSunrise: true, showSunset: true, showNight: true };
const day: WeatherDay = {
  date: "2026-10-17",
  conditions: [],
  sunrise: "07:30",
  sunset: "18:00",
};
const label = (value: Date) =>
  `${value.getHours()}:${String(value.getMinutes()).padStart(2, "0")}`;

describe("getSunIndicators", () => {
  it("places sun lines and night bands by local clock time", () => {
    expect(getSunIndicators(date, day, allOn, label)).toEqual({
      sunrise: { top: "15rem", label: "7:30" },
      sunset: { top: "36rem", label: "18:00" },
      night: [
        { top: "0rem", height: "15rem" },
        { top: "36rem", height: "12rem" },
      ],
    });
  });

  it("only shades the edge that has a sun time", () => {
    const result = getSunIndicators(
      date,
      { ...day, sunset: undefined },
      allOn,
      label,
    );
    expect(result.sunset).toBeUndefined();
    expect(result.night).toEqual([{ top: "0rem", height: "15rem" }]);
  });

  it("shows nothing without sun times", () => {
    expect(getSunIndicators(date, undefined, allOn, label)).toEqual({
      night: [],
    });
    expect(
      getSunIndicators(date, { date: day.date, conditions: [] }, allOn, label),
    ).toEqual({ night: [] });
  });

  it("hides each part when its toggle is off", () => {
    const result = getSunIndicators(
      date,
      day,
      { showSunrise: false, showSunset: true, showNight: false },
      label,
    );
    expect(result.sunrise).toBeUndefined();
    expect(result.sunset).toBeDefined();
    expect(result.night).toEqual([]);
    expect(
      getSunIndicators(date, day, { ...allOn, showSunset: false }, label)
        .sunset,
    ).toBeUndefined();
  });
});
