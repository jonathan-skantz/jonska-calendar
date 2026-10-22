import { useState } from "react";
import type { ReactNode } from "react";
import { subDays } from "date-fns";
import { formatDate, formatEventTime, localDateKey } from "../../lib/calendar";
import { describeEntryRecurrence } from "../../lib/recurrence";
import { getGathering, getGatheringStart } from "../../lib/timeline";
import type { TimeFormatPreference } from "../../lib/calendar";
import type { TimelineEntry } from "../../types";
import { ReminderEditor } from "../reminders/ReminderEditor";
import type { ReminderItem } from "../reminders/ReminderEditor";
import { Drawer } from "../../components/Drawer";
import { usePreferences } from "../calendar/preferencesContext";

type SharedEventDrawerProps = {
  entry: TimelineEntry;
  onClose: () => void;
  // The recipient's own reminders; the only thing they can change here.
  reminders?: ReminderItem[];
  reminderError?: string | null;
  onAddReminder?: (offsetSeconds: number) => void;
  onRemoveReminder?: (reminder: ReminderItem) => void;
  // The recipient's own version of the event. editor replaces the details
  // while they change it.
  onCustomize?: () => void;
  onResetCustomization?: () => void;
  customizationError?: string | null;
  editor?: ReactNode;
};

const formatRange = (
  entry: TimelineEntry,
  preference: TimeFormatPreference,
) => {
  if (!entry.starts_at || !entry.ends_at) return "";
  const start = new Date(entry.starts_at);
  const end = new Date(entry.ends_at);
  if (entry.is_all_day) {
    // All-day ends are exclusive, so the last day is the one before.
    const lastDay = subDays(end, 1);
    return localDateKey(entry.starts_at) === localDateKey(lastDay.toISOString())
      ? `${formatDate(start)}, all day`
      : `${formatDate(start)} – ${formatDate(lastDay)}, all day`;
  }
  // With gathering, the approximate start applies to when people gather.
  const { seconds, isOptional, label } = getGathering(entry);
  const gathering = seconds
    ? `, ${label ?? "gather"} ${formatEventTime(getGatheringStart(entry), entry.start_is_approximate, { preference })}${isOptional ? " (optional)" : ""}`
    : "";
  const startText = `${formatDate(start)} ${formatEventTime(start, entry.start_is_approximate && !seconds, { preference })}`;
  const endTime = formatEventTime(end, entry.end_is_approximate, {
    preference,
  });
  return localDateKey(entry.starts_at) === localDateKey(entry.ends_at)
    ? `${startText} – ${endTime}${gathering}`
    : `${startText} – ${formatDate(end)} ${endTime}${gathering}`;
};

// Recipients cannot change a shared event, only their own reminders and
// their own version of it. Events of a subscribed calendar show here too,
// without reminders.
export const SharedEventDrawer = ({
  entry,
  onClose,
  reminders,
  reminderError,
  onAddReminder,
  onRemoveReminder,
  onCustomize,
  onResetCustomization,
  customizationError,
  editor,
}: SharedEventDrawerProps) => {
  const { timeFormat } = usePreferences().preferences;
  const [showOriginal, setShowOriginal] = useState(false);
  const { original } = entry;
  return (
    <Drawer
      eyebrow={
        entry.subscription
          ? `From ${entry.subscription.name}`
          : `Shared by ${entry.shared_by}`
      }
      title={entry.title}
      className="shared-event-view"
      onClose={onClose}
    >
      {customizationError && (
        <p className="error" role="alert">
          {customizationError}
        </p>
      )}
      {editor ?? (
        <>
          <dl>
            <dt>When</dt>
            <dd>{formatRange(entry, timeFormat)}</dd>
            {entry.recurrence && (
              <>
                <dt>Repeats</dt>
                <dd>{describeEntryRecurrence(entry)}</dd>
              </>
            )}
            <dt>Kind</dt>
            <dd>{entry.kind === "attendable" ? "Attend" : "Heads-up"}</dd>
            {entry.location && (
              <>
                <dt>Location</dt>
                <dd>{entry.location}</dd>
              </>
            )}
            {entry.url && (
              <>
                <dt>URL</dt>
                <dd>
                  {/* The URL comes from another user, so only web links are clickable. */}
                  {/^https?:\/\//i.test(entry.url) ? (
                    <a href={entry.url} target="_blank" rel="noreferrer">
                      {entry.url}
                    </a>
                  ) : (
                    entry.url
                  )}
                </dd>
              </>
            )}
            {entry.description && (
              <>
                <dt>Details</dt>
                <dd className="shared-event-description">
                  {entry.description}
                </dd>
              </>
            )}
            {entry.customization?.note && (
              <>
                <dt>My note</dt>
                <dd className="shared-event-description">
                  {entry.customization.note}
                </dd>
              </>
            )}
          </dl>
          {original && (
            <div className="customized-notice">
              <p>
                ✎ You changed this event for yourself.{" "}
                <button
                  className="text-button"
                  type="button"
                  aria-expanded={showOriginal}
                  onClick={() => setShowOriginal((shown) => !shown)}
                >
                  {showOriginal ? "Hide original" : "Show original"}
                </button>
              </p>
              {showOriginal && (
                <dl aria-label="Original">
                  <dt>Title</dt>
                  <dd>{original.title}</dd>
                  <dt>When</dt>
                  <dd>
                    {formatRange(
                      {
                        ...entry,
                        starts_at: original.starts_at,
                        ends_at: original.ends_at,
                      },
                      timeFormat,
                    )}
                  </dd>
                  {original.location && (
                    <>
                      <dt>Location</dt>
                      <dd>{original.location}</dd>
                    </>
                  )}
                  {original.url && (
                    <>
                      <dt>URL</dt>
                      <dd>{original.url}</dd>
                    </>
                  )}
                  {original.description && (
                    <>
                      <dt>Details</dt>
                      <dd className="shared-event-description">
                        {original.description}
                      </dd>
                    </>
                  )}
                </dl>
              )}
            </div>
          )}
          {(onCustomize || (original && onResetCustomization)) && (
            <div className="customization-actions">
              {onCustomize && (
                <button
                  className="primary-button compact"
                  type="button"
                  onClick={onCustomize}
                >
                  {original ? "Edit my version" : "Customize"}
                </button>
              )}
              {original && onResetCustomization && (
                <button
                  className="text-button"
                  type="button"
                  onClick={onResetCustomization}
                >
                  Reset to original
                </button>
              )}
            </div>
          )}
          {onAddReminder && onRemoveReminder && (
            <ReminderEditor
              reminders={reminders ?? []}
              isAllDay={Boolean(entry.is_all_day)}
              error={reminderError}
              onAdd={onAddReminder}
              onRemove={onRemoveReminder}
            />
          )}
        </>
      )}
      <p className="form-hint">
        {entry.subscription
          ? "This event comes from a subscribed calendar and changes when the calendar does."
          : `Only ${entry.shared_by} can change this event.`}
      </p>
    </Drawer>
  );
};
