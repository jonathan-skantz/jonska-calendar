import { useEffect, useRef } from "react";
import type { FormEvent } from "react";
import {
  inputDateTime,
  MINUTES_PER_HOUR,
  PREVIEW_EVENT_DURATION_MINUTES,
} from "../lib/calendar";
import type { ComposerFieldErrors } from "../lib/formValidation";
import type { Tag, TimelineEntry } from "../types";
import { EventTagPicker } from "./EventTagPicker";

type ComposerDrawerProps = {
  type: "event" | "task";
  editingItem: TimelineEntry | null;
  selectedDate: string;
  draftEventStart: string | null;
  tags: Tag[];
  selectedTagIds: string[];
  mainTagId: string | null;
  fieldErrors: ComposerFieldErrors;
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
  fieldErrors,
  onSubmit,
  onClose,
  onTypeChange,
  onSelectedTagIdsChange,
  onMainTagIdChange,
}: ComposerDrawerProps) => {
  const startsAtInput = useRef<HTMLInputElement>(null);
  const endsAtInput = useRef<HTMLInputElement>(null);

  useEffect(() => {
    if (!draftEventStart || editingItem || type !== "event") return;
    startsAtInput.current!.value = inputDateTime(draftEventStart);
    endsAtInput.current!.value = inputDateTime(
      new Date(
        new Date(draftEventStart).getTime() +
          PREVIEW_EVENT_DURATION_MINUTES * MINUTES_PER_HOUR * 60 * 1000,
      ).toISOString(),
    );
  }, [draftEventStart, editingItem, type]);

  return <div
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
            defaultValue={editingItem?.title}
            placeholder={type === "event" ? "What is happening?" : "What needs doing?"}
            required
            autoFocus
          />
        </label>
        <label className={fieldErrors.description ? "field-invalid" : ""}>
          Description
          <textarea
            name="description"
            aria-invalid={Boolean(fieldErrors.description)}
            defaultValue={editingItem?.description}
          />
        </label>
        <label className={fieldErrors.url ? "field-invalid" : ""}>
          URL
          <input
            name="url"
            aria-invalid={Boolean(fieldErrors.url)}
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
                defaultValue={editingItem?.kind ?? "informational"}
              >
                <option value="informational">Informational</option>
                <option value="attendable">Attendable</option>
              </select>
            </label>
            <label className={fieldErrors.starts_at ? "field-invalid" : ""}>
              Starts
              <input
                ref={startsAtInput}
                name="starts_at"
                aria-invalid={Boolean(fieldErrors.starts_at)}
                type="datetime-local"
                defaultValue={inputDateTime(editingItem?.starts_at) || `${selectedDate}T09:00`}
                required
              />
            </label>
            <label className={fieldErrors.ends_at ? "field-invalid" : ""}>
              Ends
              <input
                ref={endsAtInput}
                name="ends_at"
                aria-invalid={Boolean(fieldErrors.ends_at)}
                type="datetime-local"
                defaultValue={inputDateTime(editingItem?.ends_at) || `${selectedDate}T10:00`}
                required
              />
            </label>
            <label className={fieldErrors.location ? "field-invalid" : ""}>
              Location
              <input
                name="location"
                aria-invalid={Boolean(fieldErrors.location)}
                defaultValue={editingItem?.location}
                placeholder="Home, online, or somewhere else"
              />
            </label>
            <EventTagPicker
              tags={tags}
              selectedTagIds={selectedTagIds}
              mainTagId={mainTagId}
              invalid={Boolean(fieldErrors["event-tags"] || fieldErrors["main-tag"])}
              onSelectedTagIdsChange={onSelectedTagIdsChange}
              onMainTagIdChange={onMainTagIdChange}
            />
          </>
        ) : (
          <label className={fieldErrors.deadline_at ? "field-invalid" : ""}>
            Deadline
            <input
              name="deadline_at"
              aria-invalid={Boolean(fieldErrors.deadline_at)}
              type="datetime-local"
              defaultValue={
                inputDateTime(editingItem?.deadline_at) ||
                inputDateTime(draftEventStart ?? undefined) ||
                `${selectedDate}T17:00`
              }
              required
            />
          </label>
        )}
        <button className="primary-button" type="submit">
          {editingItem ? "Save changes" : "Save item"} <span>↗</span>
        </button>
      </form>
    </aside>
  </div>;
};
