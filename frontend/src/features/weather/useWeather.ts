import { useEffect, useMemo, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { api } from "../../lib/api";
import { queryKeys } from "../../lib/queryKeys";
import type { WeatherCondition, WeatherDay } from "../../types";

type Position = { lat: number; lon: number };

const STORAGE_KEY = "weather-position";
const HOUR_MS = 60 * 60 * 1000;

// The backend caches per 0.1-degree cell anyway, so a finer position would
// only leak more precision without changing the forecast.
const roundCoordinate = (value: number) => Math.round(value * 10) / 10;

const readStoredPosition = (): Position | null => {
  try {
    const parsed: unknown = JSON.parse(
      window.localStorage.getItem(STORAGE_KEY) ?? "null",
    );
    if (!parsed || typeof parsed !== "object") return null;
    const { lat, lon } = parsed as Partial<Position>;
    return typeof lat === "number" && typeof lon === "number"
      ? { lat, lon }
      : null;
  } catch {
    return null;
  }
};

const storePosition = (position: Position | null) => {
  try {
    if (position)
      window.localStorage.setItem(STORAGE_KEY, JSON.stringify(position));
    else window.localStorage.removeItem(STORAGE_KEY);
  } catch {
    // Storage is only a speed-up for the next load.
  }
};

// Weather is an optional enhancement: without location permission or when
// the request fails, this returns an empty map and the calendar shows no icons.
export const useWeather = (enabled: boolean) => {
  const [position, setPosition] = useState(readStoredPosition);

  useEffect(() => {
    if (!enabled || !("geolocation" in navigator)) return;
    navigator.geolocation.getCurrentPosition(
      ({ coords }) => {
        const next = {
          lat: roundCoordinate(coords.latitude),
          lon: roundCoordinate(coords.longitude),
        };
        storePosition(next);
        setPosition((current) =>
          current?.lat === next.lat && current.lon === next.lon
            ? current
            : next,
        );
      },
      (error) => {
        if (error.code !== error.PERMISSION_DENIED) return;
        storePosition(null);
        setPosition(null);
      },
      { maximumAge: HOUR_MS, timeout: 10_000 },
    );
  }, [enabled]);

  const query = useQuery({
    queryKey: queryKeys.weather(position?.lat ?? 0, position?.lon ?? 0),
    queryFn: () =>
      api<WeatherDay[]>(
        `/api/weather?lat=${position?.lat}&lon=${position?.lon}`,
      ),
    enabled: enabled && position !== null,
    staleTime: HOUR_MS,
    retry: 1,
  });

  return useMemo(
    () =>
      new Map<string, WeatherCondition[]>(
        (query.data ?? []).map((day) => [day.date, day.conditions]),
      ),
    [query.data],
  );
};
