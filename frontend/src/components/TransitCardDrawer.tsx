import { useState } from "react";
import { formatDate } from "../lib/calendar";
import { getTransitCardSubmission } from "../features/transitCard/transitCardDays";
import type { TransitCardDayView } from "../features/transitCard/transitCardDays";
import type { TransitCardSubmission } from "../features/transitCard/useTransitCard";
import type { TransitCardNeed } from "../types";

type TransitCardDrawerProps = {
  views: TransitCardDayView[];
  error: string;
  onClose: () => void;
  onSave: (submission: TransitCardSubmission) => void;
};

const NEED_OPTIONS: [TransitCardNeed, string][] = [
  ["yes", "Yes"],
  ["maybe", "Maybe"],
  ["no", "No"],
];

const dayHint = (view: TransitCardDayView) => {
  if (view.state === "past") return "Ended";
  if (view.state === "not_reported") return "Not reported";
  if (view.state === "conflict") return "Conflict";
  return "";
};

export const TransitCardDrawer = ({
  views,
  error,
  onClose,
  onSave,
}: TransitCardDrawerProps) => {
  const [answers, setAnswers] = useState(
    () =>
      new Map(
        views.flatMap((view) =>
          view.state !== "past" && view.day
            ? [[view.key, view.day.need] as const]
            : [],
        ),
      ),
  );
  const submission = getTransitCardSubmission(views, answers);
  const nothingNeeded =
    submission &&
    submission.needed_dates.length === 0 &&
    submission.maybe_dates.length === 0;

  const choose = (key: string, need: TransitCardNeed) =>
    setAnswers((current) => new Map(current).set(key, need));

  return (
    <div
      className="drawer-backdrop"
      onMouseDown={(event) => event.target === event.currentTarget && onClose()}
    >
      <aside className="composer drawer transit-card-editor">
        <div className="modal-heading">
          <div>
            <p className="eyebrow">Shared transit card</p>
            <h2>Which days do you need it?</h2>
          </div>
          <button className="icon-button" onClick={onClose} aria-label="Close">
            ×
          </button>
        </div>
        {error && <p className="error">{error}</p>}
        <form
          onSubmit={(event) => {
            event.preventDefault();
            if (submission) onSave(submission);
          }}
        >
          <div className="transit-card-days">
            {views.map((view) => {
              const past = view.state === "past";
              const selected = past ? view.day?.need : answers.get(view.key);
              return (
                <div
                  className={`transit-card-day ${view.state}`}
                  key={view.key}
                  role="group"
                  aria-label={formatDate(view.date)}
                >
                  <span>{formatDate(view.date)}</span>
                  <small>{dayHint(view)}</small>
                  <div className="transit-card-choices">
                    {NEED_OPTIONS.map(([need, label]) => (
                      <button
                        type="button"
                        key={need}
                        className={`${need} ${selected === need ? "selected" : ""}`}
                        aria-pressed={selected === need}
                        disabled={past}
                        onClick={() => choose(view.key, need)}
                      >
                        {label}
                      </button>
                    ))}
                  </div>
                </div>
              );
            })}
          </div>
          <p className="form-hint">
            Saving records every remaining day shown here. Days without an
            answer are reported as not needed.
          </p>
          <button
            className="primary-button"
            type="submit"
            disabled={!submission}
          >
            {nothingNeeded ? "Save – no days needed" : "Save"}
          </button>
        </form>
      </aside>
    </div>
  );
};
