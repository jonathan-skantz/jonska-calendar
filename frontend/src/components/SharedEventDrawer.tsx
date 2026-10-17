import { subDays } from "date-fns";
import { formatDate, formatEventTime, localDateKey } from "../lib/calendar";
import { getGathering, getGatheringStart } from "../lib/timeline";
import type { TimeFormatPreference } from "../lib/calendar";
import type { TimelineEntry } from "../types";

type SharedEventDrawerProps = {
  entry: TimelineEntry;
  timeFormat: TimeFormatPreference;
  onClose: () => void;
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

// Recipients cannot change a shared event, so this view has no inputs.
export const SharedEventDrawer = ({
  entry,
  timeFormat,
  onClose,
}: SharedEventDrawerProps) => (
  <div
    className="drawer-backdrop"
    onMouseDown={(event) => event.target === event.currentTarget && onClose()}
  >
    <aside className="composer drawer shared-event-view">
      <div className="modal-heading">
        <div>
          <p className="eyebrow">Shared by {entry.shared_by}</p>
          <h2>{entry.title}</h2>
        </div>
        <button className="icon-button" onClick={onClose} aria-label="Close">
          ×
        </button>
      </div>
      <dl>
        <dt>When</dt>
        <dd>{formatRange(entry, timeFormat)}</dd>
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
            <dd className="shared-event-description">{entry.description}</dd>
          </>
        )}
      </dl>
      <p className="form-hint">Only {entry.shared_by} can change this event.</p>
    </aside>
  </div>
);
