import { render } from "@testing-library/react";
import type { ReactElement } from "react";
import { DEFAULT_CALENDAR_PREFERENCES } from "../features/calendar/calendarPreferences";
import type { CalendarPreferences } from "../features/calendar/calendarPreferences";
import { PreferencesContext } from "../features/calendar/preferencesContext";

// Renders with fixed preferences. Defaults to 24-hour times so expected
// strings don't depend on the test machine's locale.
export const renderWithPreferences = (
  ui: ReactElement,
  preferences: Partial<CalendarPreferences> = { timeFormat: "24-hour" },
) =>
  render(
    <PreferencesContext
      value={{
        preferences: { ...DEFAULT_CALENDAR_PREFERENCES, ...preferences },
        updatePreferences: () => {},
      }}
    >
      {ui}
    </PreferencesContext>,
  );
