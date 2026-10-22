import { useState } from "react";
import type { ChangeEvent } from "react";
import type { Customization, TimelineEntry } from "../../types";
import { TEXT_FIELDS, customizationFrom, draftFrom } from "./customization";
import type { CustomizationDraft } from "./customization";

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
  const change = (name: keyof CustomizationDraft, value: string) =>
    setDraft((current) => ({ ...current, [name]: value }));

  return (
    <form
      className="customization-form"
      onSubmit={async (formEvent) => {
        formEvent.preventDefault();
        const customization = customizationFrom(draft);
        if (!customization) {
          setError("Move the start and end by whole minutes.");
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
        <div className="customization-shifts">
          <label>
            Move start (min)
            <input
              type="number"
              step={1}
              min={-1440}
              max={1440}
              value={draft.startShiftMinutes}
              placeholder="0"
              onChange={(changeEvent) =>
                change("startShiftMinutes", changeEvent.target.value)
              }
            />
          </label>
          <label>
            Move end (min)
            <input
              type="number"
              step={1}
              min={-1440}
              max={1440}
              value={draft.endShiftMinutes}
              placeholder="0"
              onChange={(changeEvent) =>
                change("endShiftMinutes", changeEvent.target.value)
              }
            />
          </label>
          <p className="form-hint">
            Negative is earlier. The times move with the event if it changes.
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
