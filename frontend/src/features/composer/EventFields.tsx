import { addMinutes, subDays } from "date-fns";
import { useState } from "react";
import { DateTimeField } from "../../components/DateTimeField";
import {
  DEFAULT_EVENT_DURATION_MINUTES,
  MAX_COMMUTE_MINUTES,
  MAX_GATHERING_LABEL_LENGTH,
  SECONDS_PER_MINUTE,
  inputDateTime,
} from "../../lib/calendar";
import { cx } from "../../lib/classNames";
import type { Tag, TimelineEntry } from "../../types";
import { usePreferences } from "../calendar/preferencesContext";
import { ReminderEditor } from "../reminders/ReminderEditor";
import type { ReminderItem } from "../reminders/ReminderEditor";
import { requestNotificationPermission } from "../reminders/useReminders";
import { EventTagPicker } from "../tags/EventTagPicker";
import type { FieldState } from "./fieldState";

export type TagSelection = {
  tags: Tag[];
  selectedTagIds: string[];
  mainTagId: string | null;
  onSelectedTagIdsChange: (update: (current: string[]) => string[]) => void;
  onMainTagIdChange: (tagId: string) => void;
};

// A saved event's reminders, which change on the server right away.
export type SavedReminders = {
  items: ReminderItem[];
  error?: string | null;
  onAdd: (offsetSeconds: number) => void;
  onRemove: (reminder: ReminderItem) => void;
};

type EventFieldsProps = {
  initial?: Partial<TimelineEntry>;
  // Where a new event starts when nothing else says, e.g. the clicked hour.
  draftEventStart: string | null;
  selectedDate: string;
  fields: FieldState;
  tagSelection: TagSelection;
  // Set for a saved event. A new event queues its reminders as
  // reminder_offset form values until it is created.
  savedReminders?: SavedReminders;
  draftReminderOffsets?: number[];
};

const KINDS = [
  ["attendable", "Attend"],
  ["informational", "Heads-up"],
] as const;

const APPROXIMATE_FIELDS = [
  ["start_is_approximate", "Approximate start"],
  ["end_is_approximate", "Approximate end"],
] as const;

const MINUTE_FIELDS = [
  ["gathering_minutes", "Gathering before (min)"],
  ["commute_before_minutes", "Commute before (min)"],
  ["commute_after_minutes", "Commute after (min)"],
] as const;

const minutesFrom = (seconds?: number) =>
  seconds ? String(Math.round(seconds / SECONDS_PER_MINUTE)) : "";

export const EventFields = ({
  initial,
  draftEventStart,
  selectedDate,
  fields,
  tagSelection,
  savedReminders,
  draftReminderOffsets = [],
}: EventFieldsProps) => {
  const { timeFormat } = usePreferences().preferences;
  const [isAllDay, setIsAllDay] = useState(Boolean(initial?.is_all_day));
  const [kind, setKind] = useState(initial?.kind ?? "attendable");
  const [isSkipped, setIsSkipped] = useState(Boolean(initial?.is_skipped));
  const [queuedReminderOffsets, setQueuedReminderOffsets] =
    useState<number[]>(draftReminderOffsets);
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
  const [minutes, setMinutes] = useState({
    gathering_minutes: minutesFrom(initial?.gathering_seconds),
    commute_before_minutes: minutesFrom(initial?.commute_before_seconds),
    commute_after_minutes: minutesFrom(initial?.commute_after_seconds),
  });
  const allowsCommute = kind === "attendable" && !isAllDay;
  const defaultStart =
    inputDateTime(initial?.starts_at) ||
    inputDateTime(draftEventStart ?? undefined) ||
    `${selectedDate}T09:00`;
  // All-day ends are exclusive, so the form shows the day before.
  const defaultEnd =
    (initial?.is_all_day && initial.ends_at
      ? inputDateTime(subDays(new Date(initial.ends_at), 1).toISOString())
      : inputDateTime(initial?.ends_at)) ||
    inputDateTime(
      addMinutes(
        new Date(defaultStart),
        DEFAULT_EVENT_DURATION_MINUTES,
      ).toISOString(),
    );

  return (
    <>
      {/* The UI names are "Attend" and "Heads-up"; the stored values stay
          attendable and informational. */}
      <div
        className={cx("segmented kind-toggle", fields.labelClass("kind"))}
        role="group"
        aria-label="Kind"
      >
        {KINDS.map(([value, label]) => (
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
      <label className={fields.labelClass("starts_at")}>
        Starts
        <DateTimeField
          name="starts_at"
          label="Starts"
          invalid={fields.isInvalid("starts_at")}
          describedBy={fields.describedBy("starts_at")}
          defaultValue={defaultStart}
          preference={timeFormat}
          dateOnly={isAllDay}
          required
        />
      </label>
      <label className={fields.labelClass("ends_at")}>
        {isAllDay ? "Last day" : "Ends"}
        <DateTimeField
          name="ends_at"
          label={isAllDay ? "Last day" : "Ends"}
          invalid={fields.isInvalid("ends_at")}
          describedBy={fields.describedBy("ends_at")}
          defaultValue={defaultEnd}
          preference={timeFormat}
          dateOnly={isAllDay}
          required
        />
      </label>
      {!isAllDay && (
        <div className="approximate-fields">
          {APPROXIMATE_FIELDS.map(([name, label]) => (
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
      <label className={fields.labelClass("location")}>
        Location
        <input
          name="location"
          {...fields.inputProps("location")}
          defaultValue={initial?.location}
          placeholder="Home, online, or somewhere else"
        />
      </label>
      {allowsCommute && (
        <div className="commute-fields">
          {MINUTE_FIELDS.map(([name, label]) => (
            <label key={name} className={fields.labelClass(name)}>
              {label}
              <input
                name={name}
                type="number"
                inputMode="numeric"
                min={0}
                max={MAX_COMMUTE_MINUTES}
                step={1}
                {...fields.inputProps(name)}
                value={minutes[name]}
                onChange={(event) =>
                  setMinutes((current) => ({
                    ...current,
                    [name]: event.target.value,
                  }))
                }
              />
            </label>
          ))}
          {Number(minutes.gathering_minutes) > 0 && (
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
              <label className={fields.labelClass("gathering_label")}>
                Gathering label
                <input
                  name="gathering_label"
                  maxLength={MAX_GATHERING_LABEL_LENGTH}
                  placeholder="Warm-up, Mingle…"
                  {...fields.inputProps("gathering_label")}
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
      {savedReminders ? (
        <ReminderEditor
          reminders={savedReminders.items}
          isAllDay={isAllDay}
          error={savedReminders.error}
          onAdd={savedReminders.onAdd}
          onRemove={savedReminders.onRemove}
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
              setQueuedReminderOffsets((current) => [...current, offset]);
            }}
            onRemove={(reminder) =>
              setQueuedReminderOffsets((current) =>
                current.filter((offset) => offset !== reminder.offsetSeconds),
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
        tags={tagSelection.tags}
        selectedTagIds={tagSelection.selectedTagIds}
        mainTagId={tagSelection.mainTagId}
        invalid={fields.isInvalid("event-tags") || fields.isInvalid("main-tag")}
        errorId={fields.errorId}
        onSelectedTagIdsChange={tagSelection.onSelectedTagIdsChange}
        onMainTagIdChange={tagSelection.onMainTagIdChange}
      />
    </>
  );
};
