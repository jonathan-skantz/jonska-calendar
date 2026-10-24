import { useState } from "react";
import type { FormEvent } from "react";
import { Drawer } from "../../components/Drawer";
import { TEXT_LIMITS } from "../../lib/calendar";
import type { TimelineEntry } from "../../types";
import type { ResolvedDraft } from "../quickAdd/resolveDraft";
import { EventFields } from "./EventFields";
import type { SavedReminders, TagSelection } from "./EventFields";
import { MoreDetails } from "./MoreDetails";
import { TaskFields } from "./TaskFields";
import { COMPOSER_ERROR_ID, getFieldState } from "./fieldState";
import { getComposerFieldErrors } from "./formValidation";
import type { ComposerFieldErrors } from "./formValidation";

const FIELD_ERROR_MESSAGE = "Please correct the highlighted fields.";

type ComposerDrawerProps = {
  type: "event" | "task";
  editingItem: TimelineEntry | null;
  selectedDate: string;
  draftEventStart: string | null;
  tagSelection: TagSelection;
  // The last save or delete failure.
  error?: string;
  // Whether a save is in flight.
  pending?: boolean;
  // Called with the form values once they pass validation.
  onSave: (form: FormData) => void;
  onClose: () => void;
  onTypeChange: (type: "event" | "task") => void;
  // Set only when editing a saved event the user owns.
  onShare?: () => void;
  shareCount?: number;
  savedReminders?: SavedReminders;
  // Initial values for a new event, e.g. from quick add, and notes on what
  // the draft could not fill in.
  draft?: ResolvedDraft | null;
};

// The event or task form. The fields shared by both live here; the rest are
// in EventFields and TaskFields.
export const ComposerDrawer = ({
  type,
  editingItem,
  selectedDate,
  draftEventStart,
  tagSelection,
  error: requestError,
  pending = false,
  onSave,
  onClose,
  onTypeChange,
  onShare,
  shareCount = 0,
  savedReminders,
  draft,
}: ComposerDrawerProps) => {
  const initial = editingItem ?? draft?.values;
  const draftWarnings = draft?.warnings ?? [];
  const [fieldErrors, setFieldErrors] = useState<ComposerFieldErrors>({});
  const error =
    Object.keys(fieldErrors).length > 0 ? FIELD_ERROR_MESSAGE : requestError;
  const fields = getFieldState(fieldErrors, Boolean(error));
  const submit = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    const errors = getComposerFieldErrors(
      event.currentTarget,
      type,
      tagSelection.selectedTagIds,
      tagSelection.mainTagId,
    );
    setFieldErrors(errors);
    if (Object.keys(errors).length === 0) {
      onSave(new FormData(event.currentTarget));
    }
  };

  const descriptionAndUrl = (
    <>
      <label className={fields.labelClass("description")}>
        Description
        <textarea
          name="description"
          {...fields.inputProps("description")}
          maxLength={TEXT_LIMITS.description}
          defaultValue={initial?.description}
        />
      </label>
      <label className={fields.labelClass("url")}>
        URL
        <input
          name="url"
          {...fields.inputProps("url")}
          maxLength={TEXT_LIMITS.url}
          type="url"
          defaultValue={initial?.url}
          placeholder="https://"
        />
      </label>
    </>
  );

  return (
    <Drawer
      eyebrow={editingItem ? "Edit item" : "New item"}
      title={editingItem ? "Update the details." : "Give it a place."}
      onClose={onClose}
    >
      {error && (
        <p className="error" id={COMPOSER_ERROR_ID} role="alert">
          {error}
        </p>
      )}
      {!editingItem && (
        <div className="segmented">
          <button
            type="button"
            className={type === "event" ? "selected" : ""}
            onClick={() => onTypeChange("event")}
          >
            Event
          </button>
          <button
            type="button"
            className={type === "task" ? "selected" : ""}
            onClick={() => onTypeChange("task")}
          >
            Task
          </button>
        </div>
      )}
      {draftWarnings.length > 0 && (
        <ul className="draft-warnings" aria-label="Quick add notes">
          {draftWarnings.map((warning, index) => (
            <li key={index}>{warning}</li>
          ))}
        </ul>
      )}
      <form className="composer-form" noValidate onSubmit={submit}>
        <label className={fields.labelClass("title")}>
          Title
          <input
            name="title"
            {...fields.inputProps("title")}
            maxLength={TEXT_LIMITS.title}
            defaultValue={initial?.title}
            placeholder={
              type === "event" ? "What is happening?" : "What needs doing?"
            }
            required
            autoFocus
          />
        </label>
        {type === "event" ? (
          <EventFields
            initial={initial}
            draftEventStart={draftEventStart}
            selectedDate={selectedDate}
            fields={fields}
            tagSelection={tagSelection}
            savedReminders={savedReminders}
            draftReminderOffsets={draft?.reminderOffsets}
            moreDetails={descriptionAndUrl}
          />
        ) : (
          <>
            <TaskFields
              initial={initial}
              draftEventStart={draftEventStart}
              selectedDate={selectedDate}
              fields={fields}
            />
            <MoreDetails
              defaultOpen={Boolean(initial?.description || initial?.url)}
              hasError={
                fields.isInvalid("description") || fields.isInvalid("url")
              }
            >
              {descriptionAndUrl}
            </MoreDetails>
          </>
        )}
        {onShare && (
          <button
            className="text-button share-open-button"
            type="button"
            onClick={onShare}
          >
            {shareCount > 0
              ? `Shared with ${shareCount} – manage sharing…`
              : "Share…"}
          </button>
        )}
        <button className="primary-button" type="submit" disabled={pending}>
          {editingItem ? "Save changes" : "Save item"} <span>↗</span>
        </button>
      </form>
    </Drawer>
  );
};
