import { CALENDAR_GRID_SCALE, MINUTES_PER_HOUR } from "../../lib/calendar";
import type { WeatherDay } from "../../types";
import type { SunDisplayPreferences } from "../calendar/calendarPreferences";

const MINUTES_PER_DAY = 24 * MINUTES_PER_HOUR;

export type SunLine = { top: string; label: string };
export type NightBand = { top: string; height: string };

export type SunIndicators = {
  sunrise?: SunLine;
  sunset?: SunLine;
  night: NightBand[];
};

const toRem = (minutes: number) =>
  `${(minutes / MINUTES_PER_HOUR) * CALENDAR_GRID_SCALE}rem`;

// Sun times are local wall-clock "HH:MM" strings, so they are read directly
// instead of through Date, which would apply a time zone.
const parseClock = (value?: string) => {
  const match = value?.match(/^(\d{2}):(\d{2})$/);
  if (!match) return undefined;
  return Number(match[1]) * MINUTES_PER_HOUR + Number(match[2]);
};

export const getSunIndicators = (
  date: Date,
  day: WeatherDay | undefined,
  preferences: SunDisplayPreferences,
  formatTimeLabel: (value: Date) => string,
): SunIndicators => {
  const sunrise = parseClock(day?.sunrise);
  const sunset = parseClock(day?.sunset);
  const line = (minutes: number): SunLine => {
    const value = new Date(date);
    value.setHours(
      Math.floor(minutes / MINUTES_PER_HOUR),
      minutes % MINUTES_PER_HOUR,
      0,
      0,
    );
    return { top: toRem(minutes), label: formatTimeLabel(value) };
  };

  // Without either time the day is polar day or night (or outside the
  // forecast), which cannot be told apart here, so nothing is shaded.
  const night: NightBand[] = [];
  if (preferences.showNight) {
    if (sunrise !== undefined && sunrise > 0)
      night.push({ top: toRem(0), height: toRem(sunrise) });
    if (sunset !== undefined && sunset < MINUTES_PER_DAY)
      night.push({
        top: toRem(sunset),
        height: toRem(MINUTES_PER_DAY - sunset),
      });
  }

  return {
    sunrise:
      preferences.showSunrise && sunrise !== undefined
        ? line(sunrise)
        : undefined,
    sunset:
      preferences.showSunset && sunset !== undefined ? line(sunset) : undefined,
    night,
  };
};
