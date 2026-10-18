import type { ReactNode } from "react";
import { PreferencesContext } from "./preferencesContext";
import { useCalendarPreferences } from "./useCalendarPreferences";

// Provides the signed-in user's browser-stored calendar preferences.
export const PreferencesProvider = ({
  userId,
  children,
}: {
  userId: string;
  children: ReactNode;
}) => (
  <PreferencesContext value={useCalendarPreferences(userId)}>
    {children}
  </PreferencesContext>
);
