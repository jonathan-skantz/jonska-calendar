import { addMonths } from "date-fns";
import { useState } from "react";
import { dateKey } from "../../lib/calendar";
import { cx } from "../../lib/classNames";
import type { Recurrence, Weekday } from "../../types";
import type { FieldState } from "./fieldState";
import {
  MAX_REPEAT_COUNT,
  MAX_REPEAT_INTERVAL,
  WEEKDAYS,
  monthlyOptions,
  weekdayLabel,
  weekdayOf,
} from "../../lib/recurrence";

type RepeatFieldsProps = {
  initial?: Recurrence;
  // The event's start date; weekly and monthly repeats follow from it.
  start: Date;
  fields: FieldState;
};

type Frequency = Recurrence["frequency"] | "none";

const FREQUENCIES: [Frequency, string][] = [
  ["none", "Does not repeat"],
  ["daily", "Daily"],
  ["weekly", "Weekly"],
  ["monthly", "Monthly"],
  ["yearly", "Yearly"],
];

const UNITS: Record<Recurrence["frequency"], string> = {
  daily: "days",
  weekly: "weeks",
  monthly: "months",
  yearly: "years",
};

type Ends = "never" | "on" | "after";

const DEFAULT_COUNT = "10";

// How an event repeats. The values travel as repeat_* form fields, which
// readRecurrence turns into the rule.
export const RepeatFields = ({ initial, start, fields }: RepeatFieldsProps) => {
  const [frequency, setFrequency] = useState<Frequency>(
    initial?.frequency ?? "none",
  );
  const [interval, setInterval] = useState(String(initial?.interval ?? 1));
  const [weekdays, setWeekdays] = useState<Weekday[]>(initial?.weekdays ?? []);
  const [monthlyBy, setMonthlyBy] = useState(
    initial?.monthly_by ?? "day_of_month",
  );
  const [ends, setEnds] = useState<Ends>(
    initial?.until ? "on" : initial?.count ? "after" : "never",
  );
  const [until, setUntil] = useState(
    initial?.until ?? dateKey(addMonths(start, 3)),
  );
  const [count, setCount] = useState(String(initial?.count ?? DEFAULT_COUNT));

  // The start's weekday always repeats, and a monthly day the start no
  // longer allows falls back to the same day of the month.
  const startDay = weekdayOf(start);
  const selectedWeekdays = WEEKDAYS.filter(
    (day) => day === startDay || weekdays.includes(day),
  );
  const monthly = monthlyOptions(start);
  const selectedMonthlyBy = monthly.some(({ value }) => value === monthlyBy)
    ? monthlyBy
    : "day_of_month";

  return (
    <div className="repeat-fields">
      <label>
        Repeat
        <select
          name="repeat_frequency"
          value={frequency}
          onChange={(event) => setFrequency(event.target.value as Frequency)}
        >
          {FREQUENCIES.map(([value, label]) => (
            <option key={value} value={value}>
              {label}
            </option>
          ))}
        </select>
      </label>
      {frequency !== "none" && (
        <>
          <label className={fields.labelClass("repeat_interval")}>
            Every
            <span className="repeat-interval">
              <input
                name="repeat_interval"
                aria-label={`Repeat every how many ${UNITS[frequency]}`}
                type="number"
                inputMode="numeric"
                min={1}
                max={MAX_REPEAT_INTERVAL}
                step={1}
                required
                {...fields.inputProps("repeat_interval")}
                value={interval}
                onChange={(event) => setInterval(event.target.value)}
              />
              {UNITS[frequency]}
            </span>
          </label>
          {frequency === "weekly" && (
            <div className="weekday-chips" role="group" aria-label="On">
              {WEEKDAYS.map((day) => {
                const selected = selectedWeekdays.includes(day);
                return (
                  <button
                    key={day}
                    type="button"
                    className={cx(selected && "selected")}
                    aria-pressed={selected}
                    // The start's own weekday cannot be left out.
                    disabled={day === startDay}
                    onClick={() =>
                      setWeekdays(
                        selected
                          ? selectedWeekdays.filter((other) => other !== day)
                          : [...selectedWeekdays, day],
                      )
                    }
                  >
                    {weekdayLabel(day)}
                  </button>
                );
              })}
              {selectedWeekdays.map((day) => (
                <input
                  key={day}
                  type="hidden"
                  name="repeat_weekday"
                  value={day}
                />
              ))}
            </div>
          )}
          {frequency === "monthly" && (
            <label>
              Day of the month
              <select
                name="repeat_monthly_by"
                value={selectedMonthlyBy}
                onChange={(event) =>
                  setMonthlyBy(
                    event.target.value as NonNullable<Recurrence["monthly_by"]>,
                  )
                }
              >
                {monthly.map(({ value, label }) => (
                  <option key={value} value={value}>
                    {label}
                  </option>
                ))}
              </select>
            </label>
          )}
          <label>
            Ends
            <select
              name="repeat_ends"
              value={ends}
              onChange={(event) => setEnds(event.target.value as Ends)}
            >
              <option value="never">Never</option>
              <option value="on">On a date</option>
              <option value="after">After a number of times</option>
            </select>
          </label>
          {ends === "on" && (
            <label className={fields.labelClass("repeat_until")}>
              Last date
              <input
                name="repeat_until"
                type="date"
                min={dateKey(start)}
                required
                {...fields.inputProps("repeat_until")}
                value={until}
                onChange={(event) => setUntil(event.target.value)}
              />
            </label>
          )}
          {ends === "after" && (
            <label className={fields.labelClass("repeat_count")}>
              Times
              <input
                name="repeat_count"
                type="number"
                inputMode="numeric"
                min={1}
                max={MAX_REPEAT_COUNT}
                step={1}
                required
                {...fields.inputProps("repeat_count")}
                value={count}
                onChange={(event) => setCount(event.target.value)}
              />
            </label>
          )}
        </>
      )}
    </div>
  );
};
