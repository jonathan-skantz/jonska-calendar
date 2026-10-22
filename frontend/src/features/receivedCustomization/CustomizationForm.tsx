import { useState } from "react";
import type { ChangeEvent } from "react";
import type { Customization, TimelineEntry } from "../../types";
import { TEXT_FIELDS, customizationFrom, draftFrom } from "./customization";
import type { CustomizationDraft, MinutesField } from "./customization";

type CustomizationFormProps = {
  entry: TimelineEntry;
  onSave: (customization: Customization) => Promise<string | null>;
  onCancel: () => void;
};

// The recipient's own version of an event they receive. Each blank field
// shows what they received, and keeps following it.
export const CustomizationForm = ({
  entry,
  onSave,
  onCancel,
}: CustomizationFormProps) => {
  const received = entry.original ?? entry;
  const [draft, setDraft] = useState(() => draftFrom(entry.customization));
  const [error, setError] = useState<string | null>(null);
  const change = <Key extends keyof CustomizationDraft>(
    name: Key,
    value: CustomizationDraft[Key],
  ) => setDraft((current) => ({ ...current, [name]: value }));
  const minutesField = (
    name: MinutesField,
    label: string,
    { min = 0, placeholder = "0" } = {},
  ) => (
    <label>
      {label}
      <input
        type="number"
        step={1}
        min={min}
        max={1440}
        value={draft[name]}
        placeholder={placeholder}
        onChange={(changeEvent) => change(name, changeEvent.target.value)}
      />
    </label>
  );
  // Commute and gathering only apply to timed events the user attends.
  const attends = entry.kind === "attendable" && !entry.is_all_day;
  const hasGathering =
    draft.gatheringMinutes.trim() !== "" && Number(draft.gatheringMinutes) > 0;

  return (
    <form
      className="customization-form"
      onSubmit={async (formEvent) => {
        formEvent.preventDefault();
        const customization = customizationFrom(draft);
        if (!customization) {
          setError(
            "Use whole minutes; only the start and end can move earlier.",
          );
          return;
        }
        const saveError = await onSave(customization);
        setError(saveError);
        if (!saveError) onCancel();
      }}
    >
      {error && (
        <p className="error" role="alert">
          {error}
        </p>
      )}
      {TEXT_FIELDS.map(({ name, label, ...field }) => {
        const props = {
          name,
          value: draft[name],
          placeholder:
            (name === "note" ? undefined : received[name]) || "Not set",
          onChange: (
            changeEvent: ChangeEvent<HTMLInputElement | HTMLTextAreaElement>,
          ) => change(name, changeEvent.target.value),
        };
        return (
          <label key={name}>
            {label}
            {"multiline" in field ? (
              <textarea {...props} />
            ) : (
              <input {...props} type={name === "url" ? "url" : "text"} />
            )}
          </label>
        );
      })}
      {!entry.is_all_day && (
        <div className="customization-numbers">
          {minutesField("startShiftMinutes", "Move start (min)", {
            min: -1440,
          })}
          {minutesField("endShiftMinutes", "Move end (min)", { min: -1440 })}
          <p className="form-hint">
            Negative is earlier. The times move with the event if it changes.
          </p>
        </div>
      )}
      {attends && (
        <div className="customization-numbers">
          {minutesField("commuteBeforeMinutes", "Commute before (min)")}
          {minutesField("commuteAfterMinutes", "Commute after (min)")}
          <p className="form-hint">Your commute is only for you.</p>
          {minutesField("gatheringMinutes", "Gathering (min)", {
            placeholder: received.gathering_seconds
              ? String(received.gathering_seconds / 60)
              : "None",
          })}
          <label>
            Gathering label
            <input
              type="text"
              maxLength={40}
              value={draft.gatheringLabel}
              placeholder={received.gathering_label || "Not set"}
              disabled={!hasGathering}
              onChange={(changeEvent) =>
                change("gatheringLabel", changeEvent.target.value)
              }
            />
          </label>
          <label className="all-day-toggle">
            <input
              type="checkbox"
              checked={draft.gatheringIsOptional}
              disabled={!hasGathering}
              onChange={(changeEvent) =>
                change("gatheringIsOptional", changeEvent.target.checked)
              }
            />
            Optional gathering
          </label>
          <p className="form-hint">
            Blank keeps the shared gathering; 0 removes it.
          </p>
        </div>
      )}
      <div className="customization-actions">
        <button className="primary-button compact" type="submit">
          Save my version
        </button>
        <button className="text-button" type="button" onClick={onCancel}>
          Cancel
        </button>
      </div>
    </form>
  );
};
