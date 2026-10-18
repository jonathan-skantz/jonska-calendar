import { useState } from "react";
import type { ReactNode } from "react";
import type {
  SunDisplayPreferences,
  WeekStartPreference,
} from "../calendar/calendarPreferences";
import { weekdayName } from "../calendar/calendarPreferences";
import type { TimeFormatPreference, WeekStartsOn } from "../../lib/calendar";
import { usePreferences } from "../calendar/preferencesContext";

type SettingsPageProps = {
  displayName: string;
  displayNamePlaceholder: string;
  onDisplayNameSave: (value: string) => Promise<string | null>;
  onBack: () => void;
  children?: ReactNode;
};

export function SettingsPage({
  displayName,
  displayNamePlaceholder,
  onDisplayNameSave,
  onBack,
  children,
}: SettingsPageProps) {
  const { preferences, updatePreferences } = usePreferences();
  const [displayNameMessage, setDisplayNameMessage] = useState("");
  const weekOptions: { value: WeekStartPreference; label: string }[] = [
    { value: "browser", label: "Use browser setting" },
    ...([0, 1, 2, 3, 4, 5, 6] as WeekStartsOn[]).map((day) => ({
      value: day,
      label: weekdayName(day),
    })),
  ];

  return (
    <main className="settings-page">
      <button className="text-button settings-back" onClick={onBack}>
        ← Back to calendar
      </button>
      <h1>Settings</h1>

      <section
        className="settings-section"
        aria-labelledby="display-name-heading"
      >
        <h2 id="display-name-heading">Display name</h2>
        <p>Shown to your family, for example on transit card days.</p>
        <form
          className="settings-inline-form"
          onSubmit={async (event) => {
            event.preventDefault();
            const value = String(
              new FormData(event.currentTarget).get("display-name") ?? "",
            );
            const error = await onDisplayNameSave(value);
            setDisplayNameMessage(error ?? "Saved.");
          }}
        >
          <label className="settings-select-label" htmlFor="display-name">
            Name
          </label>
          <input
            id="display-name"
            name="display-name"
            defaultValue={displayName}
            placeholder={displayNamePlaceholder}
            maxLength={60}
          />
          <button className="primary-button compact" type="submit">
            Save
          </button>
          {displayNameMessage && (
            <span className="form-hint" role="status">
              {displayNameMessage}
            </span>
          )}
        </form>
      </section>

      {children}

      <section
        className="settings-section"
        aria-labelledby="time-format-heading"
      >
        <h2 id="time-format-heading">Time format</h2>
        <div
          className="settings-options"
          role="group"
          aria-labelledby="time-format-heading"
        >
          {(
            [
              ["browser", "Use browser setting"],
              ["12-hour", "12-hour (AM/PM)"],
              ["24-hour", "24-hour"],
            ] as [TimeFormatPreference, string][]
          ).map(([value, label]) => (
            <label className="settings-option" key={value}>
              <input
                type="radio"
                name="time-format"
                value={value}
                checked={preferences.timeFormat === value}
                onChange={() => updatePreferences({ timeFormat: value })}
              />
              <span>{label}</span>
            </label>
          ))}
        </div>
      </section>

      <section
        className="settings-section"
        aria-labelledby="week-start-heading"
      >
        <h2 id="week-start-heading">Start week on</h2>
        <label className="settings-select-label" htmlFor="week-start">
          First day of the week
        </label>
        <select
          id="week-start"
          value={preferences.weekStartsOn}
          onChange={(event) =>
            updatePreferences({
              weekStartsOn:
                event.target.value === "browser"
                  ? "browser"
                  : (Number(event.target.value) as WeekStartsOn),
            })
          }
        >
          {weekOptions.map((option) => (
            <option key={option.value} value={option.value}>
              {option.label}
            </option>
          ))}
        </select>
      </section>

      <section className="settings-section" aria-labelledby="sun-heading">
        <h2 id="sun-heading">Sun and night</h2>
        <p>Uses the forecast for your current location.</p>
        <div className="settings-options">
          {(
            [
              ["showSunrise", "Show sunrise"],
              ["showSunset", "Show sunset"],
              ["showNight", "Shade night"],
            ] as [keyof SunDisplayPreferences, string][]
          ).map(([key, label]) => (
            <label className="settings-option" key={key}>
              <input
                type="checkbox"
                checked={preferences[key]}
                onChange={(event) =>
                  updatePreferences({ [key]: event.target.checked })
                }
              />
              <span>{label}</span>
            </label>
          ))}
        </div>
      </section>
    </main>
  );
}
