import { formatDate, formatTime } from "../../lib/calendar";
import { cx } from "../../lib/classNames";
import type {
  AssistantAction,
  AssistantActionKind,
  AssistantActionStatus,
  AssistantDecision,
} from "../../types";
import { usePreferences } from "../calendar/preferencesContext";

const KIND_LABELS: Record<AssistantActionKind, string> = {
  create: "New event",
  move: "Move to",
  skip: "Skip",
  unskip: "Attend again",
  delete: "Delete",
};

const STATUS_LABELS: Partial<Record<AssistantActionStatus, string>> = {
  applied: "Applied",
  declined: "Declined",
  undone: "Undone",
};

type ActionCardProps = {
  action: AssistantAction;
  busy: boolean;
  // Why the last decision on this action failed.
  error?: string;
  onDecide: (decision: AssistantDecision) => void;
  onOpen: () => void;
};

// One change the assistant made or asks for. Created events can be opened or
// undone; other changes wait for Apply or Decline.
export const ActionCard = ({
  action,
  busy,
  error,
  onDecide,
  onOpen,
}: ActionCardProps) => {
  const { timeFormat } = usePreferences().preferences;
  const day = formatDate(new Date(action.starts_at));
  const time = (value: string) => formatTime(value, { preference: timeFormat });
  const when = action.is_all_day
    ? `${day}, all day`
    : `${day} ${time(action.starts_at)}–${time(action.ends_at)}`;
  const button = (decision: AssistantDecision, label: string) => (
    <button
      type="button"
      className={decision === "apply" ? "primary-button" : "text-button"}
      disabled={busy}
      onClick={() => onDecide(decision)}
    >
      {label}
    </button>
  );

  return (
    <div className={cx("assistant-action", action.status)}>
      <span className="assistant-action-kind">{KIND_LABELS[action.kind]}</span>
      <strong>{action.title}</strong>
      <span>{when}</span>
      {action.warnings?.map((warning) => (
        <span key={warning} className="assistant-warning">
          {warning}
        </span>
      ))}
      {error && (
        <span className="error" role="alert">
          {error}
        </span>
      )}
      <div className="assistant-action-buttons">
        {action.status === "done" && (
          <>
            <button type="button" className="text-button" onClick={onOpen}>
              Open
            </button>
            {button("undo", "Undo")}
          </>
        )}
        {action.status === "pending" && (
          <>
            {button("apply", "Apply")}
            {button("decline", "Decline")}
          </>
        )}
        {STATUS_LABELS[action.status] && (
          <span className="assistant-status">
            {STATUS_LABELS[action.status]}
          </span>
        )}
      </div>
    </div>
  );
};
