import { useState } from "react";
import {
  DEFAULT_CALENDAR_PREFERENCES,
  parseCalendarPreferences,
} from "./calendarPreferences";
import type { CalendarPreferences } from "./calendarPreferences";

const storageKey = (userId: string) => `calendar-preferences:${userId}`;

export const useCalendarPreferences = (userId: string) => {
  const [stored, setStored] = useState(() => {
    let value: string | null = null;
    try {
      value = window.localStorage.getItem(storageKey(userId));
    } catch {
      // Keep browser defaults when storage is unavailable.
    }
    return { userId, preferences: parseCalendarPreferences(value) };
  });
  const preferences =
    stored.userId === userId
      ? stored.preferences
      : DEFAULT_CALENDAR_PREFERENCES;

  const updatePreferences = (update: Partial<CalendarPreferences>) => {
    const next = { ...preferences, ...update };
    setStored({ userId, preferences: next });
    try {
      window.localStorage.setItem(storageKey(userId), JSON.stringify(next));
    } catch {
      // The selected values still apply for this session.
    }
  };

  return { preferences, updatePreferences };
};
