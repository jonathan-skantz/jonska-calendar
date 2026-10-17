import { addMinutes, subDays } from "date-fns";
import { useState } from "react";
import type { FormEvent } from "react";
import {
  DEFAULT_EVENT_DURATION_MINUTES,
  MAX_COMMUTE_MINUTES,
  MAX_GATHERING_LABEL_LENGTH,
  SECONDS_PER_MINUTE,
  inputDateTime,
  type TimeFormatPreference,
} from "../../lib/calendar";
import type { ComposerFieldErrors } from "./formValidation";
import type { Tag, TimelineEntry } from "../../types";
import { EventTagPicker } from "../tags/EventTagPicker";
import { DateTimeField } from "../../components/DateTimeField";
import { ReminderEditor } from "../reminders/ReminderEditor";
import type { ReminderItem } from "../reminders/ReminderEditor";
import { requestNotificationPermission } from "../reminders/useReminders";

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
  // Set only when editing a saved event the user owns.
  onShare?: () => void;
  // The saved event's reminders, changed right away. A new event queues its
  // reminders as reminder_offset form values until it is created.
  reminders?: ReminderItem[];
  reminderError?: string | null;
  onAddReminder?: (offsetSeconds: number) => void;
  onRemoveReminder?: (reminder: ReminderItem) => void;
  shareCount?: number;
  // Initial values for a new event, e.g. from quick add, and notes on what
  // the draft could not fill in.
  draftValues?: Partial<TimelineEntry>;
  draftReminderOffsets?: number[];
  draftWarnings?: string[];
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
  onShare,
  shareCount = 0,
  reminders = [],
  reminderError,
  onAddReminder,
  onRemoveReminder,
  draftValues,
  draftReminderOffsets = [],
  draftWarnings = [],
}: ComposerDrawerProps) => {
  const initial = editingItem ?? draftValues;
  const [isAllDay, setIsAllDay] = useState(Boolean(initial?.is_all_day));
  const [kind, setKind] = useState(initial?.kind ?? "attendable");
  const isSavedEvent = editingItem?.type === "event";
  const [queuedReminderOffsets, setQueuedReminderOffsets] =
    useState<number[]>(draftReminderOffsets);
  const [isSkipped, setIsSkipped] = useState(Boolean(initial?.is_skipped));
  const allowsCommute = kind === "attendable" && !isAllDay;
  const commuteMinutes = (seconds?: number) =>
    seconds ? String(Math.round(seconds / SECONDS_PER_MINUTE)) : "";
  // Kept in state so values survive while the fields are hidden by a kind or
  // all-day toggle.
  const [approximate, setApproximate] = useState({
    start_is_approximate: Boolean(initial?.start_is_approximate),
    end_is_approximate: Boolean(initial?.end_is_approximate),
  });
  const [gatheringDetails, setGatheringDetails] = useState({
    gathering_is_optional: Boolean(initial?.gathering_is_optional),
    gathering_label: initial?.gathering_label ?? "",
  });
  const [commute, setCommute] = useState({
    gathering_minutes: commuteMinutes(initial?.gathering_seconds),
    commute_before_minutes: commuteMinutes(initial?.commute_before_seconds),
    commute_after_minutes: commuteMinutes(initial?.commute_after_seconds),
  });
  const defaultEventStart =
    inputDateTime(initial?.starts_at) ||
    inputDateTime(draftEventStart ?? undefined) ||
    `${selectedDate}T09:00`;
  const defaultEventEnd =
    (initial?.is_all_day && initial.ends_at
      ? inputDateTime(subDays(new Date(initial.ends_at), 1).toISOString())
      : inputDateTime(initial?.ends_at)) ||
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
        {draftWarnings.length > 0 && (
          <ul className="draft-warnings" aria-label="Quick add notes">
            {draftWarnings.map((warning, index) => (
              <li key={index}>{warning}</li>
            ))}
          </ul>
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
              defaultValue={initial?.title}
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
              defaultValue={initial?.description}
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
              defaultValue={initial?.url}
              placeholder="https://"
            />
          </label>
          {type === "event" ? (
            <>
              {/* The UI names are "Attend" and "Heads-up"; the stored values
                  stay attendable and informational. */}
              <div
                className={`segmented kind-toggle ${fieldErrors.kind ? "field-invalid" : ""}`}
                role="group"
                aria-label="Kind"
              >
                {(
                  [
                    ["attendable", "Attend"],
                    ["informational", "Heads-up"],
                  ] as const
                ).map(([value, label]) => (
                  <button
                    key={value}
                    type="button"
                    className={kind === value ? "selected" : ""}
                    aria-pressed={kind === value}
                    onClick={() => setKind(value)}
                  >
                    {label}
                  </button>
                ))}
              </div>
              <input type="hidden" name="kind" value={kind} />
              {kind === "attendable" && (
                <label className="all-day-toggle">
                  <input
                    type="checkbox"
                    name="is_skipped"
                    value="true"
                    checked={isSkipped}
                    onChange={(event) => setIsSkipped(event.target.checked)}
                  />
                  Skipping this one (keep in calendar)
                </label>
              )}
              <label className="all-day-toggle">
                <input
                  type="checkbox"
                  checked={isAllDay}
                  onChange={(event) => setIsAllDay(event.target.checked)}
                />
                All day
              </label>
              <input type="hidden" name="is_all_day" value={String(isAllDay)} />
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
                  dateOnly={isAllDay}
                  required
                />
              </label>
              <label className={fieldErrors.ends_at ? "field-invalid" : ""}>
                {isAllDay ? "Last day" : "Ends"}
                <DateTimeField
                  name="ends_at"
                  label={isAllDay ? "Last day" : "Ends"}
                  invalid={Boolean(fieldErrors.ends_at)}
                  describedBy={
                    fieldErrors.ends_at && error ? "composer-error" : undefined
                  }
                  defaultValue={defaultEventEnd}
                  preference={timeFormat}
                  dateOnly={isAllDay}
                  required
                />
              </label>
              {!isAllDay && (
                <div className="approximate-fields">
                  {(
                    [
                      ["start_is_approximate", "Approximate start"],
                      ["end_is_approximate", "Approximate end"],
                    ] as const
                  ).map(([name, label]) => (
                    <label key={name} className="all-day-toggle">
                      <input
                        type="checkbox"
                        name={name}
                        value="true"
                        checked={approximate[name]}
                        onChange={(event) =>
                          setApproximate((current) => ({
                            ...current,
                            [name]: event.target.checked,
                          }))
                        }
                      />
                      {label}
                    </label>
                  ))}
                </div>
              )}
              <label className={fieldErrors.location ? "field-invalid" : ""}>
                Location
                <input
                  name="location"
                  aria-invalid={Boolean(fieldErrors.location)}
                  aria-describedby={
                    fieldErrors.location && error ? "composer-error" : undefined
                  }
                  defaultValue={initial?.location}
                  placeholder="Home, online, or somewhere else"
                />
              </label>
              {allowsCommute && (
                <div className="commute-fields">
                  {(
                    [
                      ["gathering_minutes", "Gathering before (min)"],
                      ["commute_before_minutes", "Commute before (min)"],
                      ["commute_after_minutes", "Commute after (min)"],
                    ] as const
                  ).map(([name, label]) => (
                    <label
                      key={name}
                      className={fieldErrors[name] ? "field-invalid" : ""}
                    >
                      {label}
                      <input
                        name={name}
                        type="number"
                        inputMode="numeric"
                        min={0}
                        max={MAX_COMMUTE_MINUTES}
                        step={1}
                        aria-invalid={Boolean(fieldErrors[name])}
                        aria-describedby={
                          fieldErrors[name] && error
                            ? "composer-error"
                            : undefined
                        }
                        value={commute[name]}
                        onChange={(event) =>
                          setCommute((current) => ({
                            ...current,
                            [name]: event.target.value,
                          }))
                        }
                      />
                    </label>
                  ))}
                  {Number(commute.gathering_minutes) > 0 && (
                    <div className="gathering-details">
                      <label className="all-day-toggle">
                        <input
                          type="checkbox"
                          name="gathering_is_optional"
                          value="true"
                          checked={gatheringDetails.gathering_is_optional}
                          onChange={(event) =>
                            setGatheringDetails((current) => ({
                              ...current,
                              gathering_is_optional: event.target.checked,
                            }))
                          }
                        />
                        Optional (drop-in)
                      </label>
                      <label
                        className={
                          fieldErrors.gathering_label ? "field-invalid" : ""
                        }
                      >
                        Gathering label
                        <input
                          name="gathering_label"
                          maxLength={MAX_GATHERING_LABEL_LENGTH}
                          placeholder="Warm-up, Mingle…"
                          aria-invalid={Boolean(fieldErrors.gathering_label)}
                          aria-describedby={
                            fieldErrors.gathering_label && error
                              ? "composer-error"
                              : undefined
                          }
                          value={gatheringDetails.gathering_label}
                          onChange={(event) =>
                            setGatheringDetails((current) => ({
                              ...current,
                              gathering_label: event.target.value,
                            }))
                          }
                        />
                      </label>
                    </div>
                  )}
                </div>
              )}
              {isSavedEvent ? (
                <ReminderEditor
                  reminders={reminders}
                  isAllDay={isAllDay}
                  error={reminderError}
                  onAdd={(offset) => onAddReminder?.(offset)}
                  onRemove={(reminder) => onRemoveReminder?.(reminder)}
                />
              ) : (
                <>
                  <ReminderEditor
                    reminders={queuedReminderOffsets.map((offset) => ({
                      id: String(offset),
                      offsetSeconds: offset,
                    }))}
                    isAllDay={isAllDay}
                    onAdd={(offset) => {
                      requestNotificationPermission();
                      setQueuedReminderOffsets((current) => [
                        ...current,
                        offset,
                      ]);
                    }}
                    onRemove={(reminder) =>
                      setQueuedReminderOffsets((current) =>
                        current.filter(
                          (offset) => offset !== reminder.offsetSeconds,
                        ),
                      )
                    }
                  />
                  {queuedReminderOffsets.map((offset) => (
                    <input
                      key={offset}
                      type="hidden"
                      name="reminder_offset"
                      value={offset}
                    />
                  ))}
                </>
              )}
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
          <button className="primary-button" type="submit">
            {editingItem ? "Save changes" : "Save item"} <span>↗</span>
          </button>
        </form>
      </aside>
    </div>
  );
};
