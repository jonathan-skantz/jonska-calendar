import { useEffect, useRef, useState } from "react";
import {
  getEffectiveTimeFormat,
  type TimeFormatPreference,
} from "../lib/calendar";

type DateTimeFieldProps = {
  name: string;
  label: string;
  defaultValue: string;
  preference: TimeFormatPreference;
  invalid?: boolean;
  describedBy?: string;
  required?: boolean;
  dateOnly?: boolean;
};

const splitValue = (value: string) => {
  const [date = "", time = "09:00"] = value.split("T");
  const [hour = "09", minute = "00"] = time.split(":");
  return { date, hour: Number(hour), minute };
};

export function DateTimeField({
  name,
  label,
  defaultValue,
  preference,
  invalid,
  describedBy,
  required,
  dateOnly = false,
}: DateTimeFieldProps) {
  const initial = splitValue(defaultValue);
  const [date, setDate] = useState(initial.date);
  const [hourOfDay, setHourOfDay] = useState(initial.hour);
  const [minute, setMinute] = useState(initial.minute);
  const wasDateOnly = useRef(dateOnly);
  const format = getEffectiveTimeFormat(preference);
  const period = hourOfDay >= 12 ? "PM" : "AM";
  const displayedHour = format === "12-hour" ? hourOfDay % 12 || 12 : hourOfDay;
  const displayedHours =
    format === "12-hour"
      ? Array.from({ length: 12 }, (_, index) => index + 1)
      : Array.from({ length: 24 }, (_, index) => index);
  const timeControls = dateOnly ? null : (
    <>
      <select
        aria-label={`${label} hour`}
        aria-invalid={invalid}
        aria-describedby={describedBy}
        value={displayedHour}
        onChange={(event) => {
          const selected = Number(event.target.value);
          setHourOfDay(
            format === "12-hour"
              ? (selected % 12) + (period === "PM" ? 12 : 0)
              : selected,
          );
        }}
      >
        {displayedHours.map((value) => (
          <option key={value} value={value}>
            {String(value).padStart(format === "24-hour" ? 2 : 1, "0")}
          </option>
        ))}
      </select>
      <span aria-hidden="true">:</span>
      <select
        aria-label={`${label} minute`}
        aria-invalid={invalid}
        aria-describedby={describedBy}
        value={minute}
        onChange={(event) => setMinute(event.target.value)}
      >
        {Array.from({ length: 60 }, (_, value) =>
          String(value).padStart(2, "0"),
        ).map((value) => (
          <option key={value} value={value}>
            {value}
          </option>
        ))}
      </select>
      {format === "12-hour" && (
        <select
          aria-label={`${label} AM/PM`}
          aria-invalid={invalid}
          aria-describedby={describedBy}
          value={period}
          onChange={(event) => {
            const isPm = event.target.value === "PM";
            setHourOfDay((hourOfDay % 12) + (isPm ? 12 : 0));
          }}
        >
          <option>AM</option>
          <option>PM</option>
        </select>
      )}
    </>
  );

  useEffect(() => {
    if (wasDateOnly.current && !dateOnly) {
      const isEnd = name === "ends_at";
      setHourOfDay(isEnd ? 23 : 0);
      setMinute(isEnd ? "59" : "00");
    }
    wasDateOnly.current = dateOnly;
  }, [dateOnly, name]);

  return (
    <div className="datetime-field">
      <input
        aria-label={`${label} date`}
        aria-invalid={invalid}
        aria-describedby={describedBy}
        type="date"
        data-field={name}
        value={date}
        required={required}
        onChange={(event) => setDate(event.target.value)}
      />
      {timeControls}
      <input
        type="hidden"
        name={name}
        value={
          date ? `${date}T${String(hourOfDay).padStart(2, "0")}:${minute}` : ""
        }
      />
    </div>
  );
}
