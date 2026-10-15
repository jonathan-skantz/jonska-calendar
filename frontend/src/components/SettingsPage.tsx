import type {
  CalendarPreferences,
  WeekStartPreference,
} from "../features/calendar/calendarPreferences";
import { weekdayName } from "../features/calendar/calendarPreferences";
import type { TimeFormatPreference, WeekStartsOn } from "../lib/calendar";

type SettingsPageProps = {
  preferences: CalendarPreferences;
  onTimeFormatChange: (value: TimeFormatPreference) => void;
  onWeekStartChange: (value: WeekStartPreference) => void;
  onBack: () => void;
};

export function SettingsPage({
  preferences,
  onTimeFormatChange,
  onWeekStartChange,
  onBack,
}: SettingsPageProps) {
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
                onChange={() => onTimeFormatChange(value)}
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
            onWeekStartChange(
              event.target.value === "browser"
                ? "browser"
                : (Number(event.target.value) as WeekStartsOn),
            )
          }
        >
          {weekOptions.map((option) => (
            <option key={option.value} value={option.value}>
              {option.label}
            </option>
          ))}
        </select>
      </section>
    </main>
  );
}
