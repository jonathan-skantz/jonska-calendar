import type { FormEvent } from "react";
import { inputDateTime } from "../lib/calendar";
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
  onSubmit,
  onClose,
  onTypeChange,
  onSelectedTagIdsChange,
  onMainTagIdChange,
}: ComposerDrawerProps) => (
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
      <form className="composer-form" onSubmit={onSubmit}>
        <label>
          Title
          <input
            name="title"
            defaultValue={editingItem?.title}
            placeholder={type === "event" ? "What is happening?" : "What needs doing?"}
            required
            autoFocus
          />
        </label>
        <label>
          Description
          <textarea name="description" defaultValue={editingItem?.description} />
        </label>
        <label>
          URL
          <input
            name="url"
            type="url"
            defaultValue={editingItem?.url}
            placeholder="https://"
          />
        </label>
        {type === "event" ? (
          <>
            <label>
              Kind
              <select name="kind" defaultValue={editingItem?.kind ?? "informational"}>
                <option value="informational">Informational</option>
                <option value="attendable">Attendable</option>
              </select>
            </label>
            <label>
              Starts
              <input
                name="starts_at"
                type="datetime-local"
                defaultValue={inputDateTime(editingItem?.starts_at) || `${selectedDate}T09:00`}
                required
              />
            </label>
            <label>
              Ends
              <input
                name="ends_at"
                type="datetime-local"
                defaultValue={inputDateTime(editingItem?.ends_at) || `${selectedDate}T10:00`}
                required
              />
            </label>
            <label>
              Location
              <input
                name="location"
                defaultValue={editingItem?.location}
                placeholder="Home, online, or somewhere else"
              />
            </label>
            <EventTagPicker
              tags={tags}
              selectedTagIds={selectedTagIds}
              mainTagId={mainTagId}
              onSelectedTagIdsChange={onSelectedTagIdsChange}
              onMainTagIdChange={onMainTagIdChange}
            />
          </>
        ) : (
          <label>
            Deadline
            <input
              name="deadline_at"
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
  </div>
);
