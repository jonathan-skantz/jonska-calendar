import { createContext, use } from "react";
import { DEFAULT_CALENDAR_PREFERENCES } from "./calendarPreferences";
import type { CalendarPreferences } from "./calendarPreferences";

type PreferencesValue = {
  preferences: CalendarPreferences;
  updatePreferences: (update: Partial<CalendarPreferences>) => void;
};

// The defaults let components render outside a provider, as in tests that
// don't care about preferences.
export const PreferencesContext = createContext<PreferencesValue>({
  preferences: DEFAULT_CALENDAR_PREFERENCES,
  updatePreferences: () => {},
});

export const usePreferences = () => use(PreferencesContext);
