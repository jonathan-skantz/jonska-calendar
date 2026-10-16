import { addMinutes } from "date-fns";
import type { FormEvent } from "react";
import {
  DEFAULT_EVENT_DURATION_MINUTES,
  inputDateTime,
  type TimeFormatPreference,
} from "../lib/calendar";
import type { ComposerFieldErrors } from "../lib/formValidation";
import type { Tag, TimelineEntry } from "../types";
import { EventTagPicker } from "./EventTagPicker";
import { DateTimeField } from "./DateTimeField";

type ComposerDrawerProps = {
  type: "event" | "task";
  editingItem: TimelineEntry | null;
  selectedDate: string;
  draftEventStart: string | null;
  tags: Tag[];
  selectedTagIds: string[];
  mainTagId: string | null;
  timeFormat: TimeFormatPreference;
  fieldErrors: ComposerFieldErrors;
  error?: string;
  onSubmit: (event: FormEvent<HTMLFormElement>) => void;
  onClose: () => void;
  onTypeChange: (type: "event" | "task") => void;
  onSelectedTagIdsChange: (update: (current: string[]) => string[]) => void;
  onMainTagIdChange: (tagId: string) => void;
};

export const ComposerDrawer = ({
  type,
  editingItem,
  selectedDate,
  draftEventStart,
  tags,
  selectedTagIds,
  mainTagId,
  timeFormat,
  fieldErrors,
  error,
  onSubmit,
  onClose,
  onTypeChange,
  onSelectedTagIdsChange,
  onMainTagIdChange,
}: ComposerDrawerProps) => {
  const defaultEventStart =
    inputDateTime(editingItem?.starts_at) ||
    inputDateTime(draftEventStart ?? undefined) ||
    `${selectedDate}T09:00`;
  const defaultEventEnd =
    inputDateTime(editingItem?.ends_at) ||
    inputDateTime(
      addMinutes(
        new Date(defaultEventStart),
        DEFAULT_EVENT_DURATION_MINUTES,
      ).toISOString(),
    );

  return (
    <div
      className="drawer-backdrop"
      onMouseDown={(event) => event.target === event.currentTarget && onClose()}
    >
      <aside
        className="composer drawer"
        key={`${editingItem?.id ?? "new"}-${type}`}
      >
        <div className="modal-heading">
          <div>
            <p className="eyebrow">{editingItem ? "Edit item" : "New item"}</p>
            <h2>{editingItem ? "Update the details." : "Give it a place."}</h2>
          </div>
          <button className="icon-button" onClick={onClose} aria-label="Close">
            ×
          </button>
        </div>
        {error && (
          <p className="error" id="composer-error" role="alert">
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
        <form className="composer-form" noValidate onSubmit={onSubmit}>
          <label className={fieldErrors.title ? "field-invalid" : ""}>
            Title
            <input
              name="title"
              aria-invalid={Boolean(fieldErrors.title)}
              aria-describedby={
                fieldErrors.title && error ? "composer-error" : undefined
              }
              defaultValue={editingItem?.title}
              placeholder={
                type === "event" ? "What is happening?" : "What needs doing?"
              }
              required
              autoFocus
            />
          </label>
          <label className={fieldErrors.description ? "field-invalid" : ""}>
            Description
            <textarea
              name="description"
              aria-invalid={Boolean(fieldErrors.description)}
              aria-describedby={
                fieldErrors.description && error ? "composer-error" : undefined
              }
              defaultValue={editingItem?.description}
            />
          </label>
          <label className={fieldErrors.url ? "field-invalid" : ""}>
            URL
            <input
              name="url"
              aria-invalid={Boolean(fieldErrors.url)}
              aria-describedby={
                fieldErrors.url && error ? "composer-error" : undefined
              }
              type="url"
              defaultValue={editingItem?.url}
              placeholder="https://"
            />
          </label>
          {type === "event" ? (
            <>
              <label className={fieldErrors.kind ? "field-invalid" : ""}>
                Kind
                <select
                  name="kind"
                  aria-invalid={Boolean(fieldErrors.kind)}
                  aria-describedby={
                    fieldErrors.kind && error ? "composer-error" : undefined
                  }
                  defaultValue={editingItem?.kind ?? "informational"}
                >
                  <option value="informational">Informational</option>
                  <option value="attendable">Attendable</option>
                </select>
              </label>
              <label className={fieldErrors.starts_at ? "field-invalid" : ""}>
                Starts
                <DateTimeField
                  name="starts_at"
                  label="Starts"
                  invalid={Boolean(fieldErrors.starts_at)}
                  describedBy={
                    fieldErrors.starts_at && error
                      ? "composer-error"
                      : undefined
                  }
                  defaultValue={defaultEventStart}
                  preference={timeFormat}
                  required
                />
              </label>
              <label className={fieldErrors.ends_at ? "field-invalid" : ""}>
                Ends
                <DateTimeField
                  name="ends_at"
                  label="Ends"
                  invalid={Boolean(fieldErrors.ends_at)}
                  describedBy={
                    fieldErrors.ends_at && error ? "composer-error" : undefined
                  }
                  defaultValue={defaultEventEnd}
                  preference={timeFormat}
                  required
                />
              </label>
              <label className={fieldErrors.location ? "field-invalid" : ""}>
                Location
                <input
                  name="location"
                  aria-invalid={Boolean(fieldErrors.location)}
                  aria-describedby={
                    fieldErrors.location && error ? "composer-error" : undefined
                  }
                  defaultValue={editingItem?.location}
                  placeholder="Home, online, or somewhere else"
                />
              </label>
              <EventTagPicker
                tags={tags}
                selectedTagIds={selectedTagIds}
                mainTagId={mainTagId}
                invalid={Boolean(
                  fieldErrors["event-tags"] || fieldErrors["main-tag"],
                )}
                errorId={error ? "composer-error" : undefined}
                onSelectedTagIdsChange={onSelectedTagIdsChange}
                onMainTagIdChange={onMainTagIdChange}
              />
            </>
          ) : (
            <label className={fieldErrors.deadline_at ? "field-invalid" : ""}>
              Deadline
              <DateTimeField
                name="deadline_at"
                label="Deadline"
                invalid={Boolean(fieldErrors.deadline_at)}
                describedBy={
                  fieldErrors.deadline_at && error
                    ? "composer-error"
                    : undefined
                }
                defaultValue={
                  inputDateTime(editingItem?.deadline_at) ||
                  inputDateTime(draftEventStart ?? undefined) ||
                  `${selectedDate}T17:00`
                }
                preference={timeFormat}
                required
              />
            </label>
          )}
          <button className="primary-button" type="submit">
            {editingItem ? "Save changes" : "Save item"} <span>↗</span>
          </button>
        </form>
      </aside>
    </div>
  );
};
