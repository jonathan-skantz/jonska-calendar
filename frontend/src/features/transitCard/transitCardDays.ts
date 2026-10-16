import { dateKey } from "../../lib/calendar";
import type { TransitCardDay, TransitCardNeed } from "../../types";

export type TransitCardDayState =
  "past" | "not_reported" | "not_needed" | "maybe" | "resolved" | "conflict";

export type TransitCardDayView = {
  date: Date;
  key: string;
  state: TransitCardDayState;
  day?: TransitCardDay;
};

// A missing row means "not reported". Days that have ended are read-only and
// are never shown as missing, but keep their recorded answer for display.
export const getTransitCardDayViews = (
  dates: Date[],
  days: TransitCardDay[],
  today: Date,
): TransitCardDayView[] => {
  const todayKey = dateKey(today);
  const byDate = new Map(days.map((day) => [day.date, day]));
  return dates.map((date) => {
    const key = dateKey(date);
    const day = byDate.get(key);
    let state: TransitCardDayState;
    if (key < todayKey) state = "past";
    else if (!day) state = "not_reported";
    else if (day.need === "no") state = "not_needed";
    else if (day.need === "maybe") state = "maybe";
    else state = day.status;
    return { date, key, state, day };
  });
};

export type TransitCardPillVariant =
  "pending" | "not_needed" | "maybe" | "resolved" | "conflict";

// Every answer gets a pill, so "no" is never confused with "not reported".
// Only days that ended without an answer get no pill.
export const getTransitCardPillVariant = (
  view: TransitCardDayView,
): TransitCardPillVariant | null => {
  if (view.state === "not_reported") return "pending";
  if (!view.day) return null;
  if (view.day.need === "no") return "not_needed";
  return view.day.need === "maybe" ? "maybe" : view.day.status;
};

// Only days that have not ended are submitted. Every one of them gets an
// explicit answer: days without a "yes" or "maybe" are sent as "no", so an
// empty selection is still a report.
export const getTransitCardSubmission = (
  views: TransitCardDayView[],
  answers: Map<string, TransitCardNeed>,
) => {
  const editable = views.filter((view) => view.state !== "past");
  if (editable.length === 0) return null;
  const keysWith = (need: TransitCardNeed) =>
    editable
      .filter((view) => answers.get(view.key) === need)
      .map((view) => view.key);
  return {
    from: editable[0].key,
    to: editable[editable.length - 1].key,
    needed_dates: keysWith("yes"),
    maybe_dates: keysWith("maybe"),
  };
};
