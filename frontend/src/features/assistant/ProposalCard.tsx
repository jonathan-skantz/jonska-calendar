import { formatDate, formatTime } from "../../lib/calendar";
import type { AssistantProposal } from "../../types";
import { usePreferences } from "../calendar/preferencesContext";

type ProposalCardProps = {
  proposal: AssistantProposal;
  onOpen: () => void;
};

// An event the assistant suggests. Opening it prefills the event form; the
// user still saves it there.
export const ProposalCard = ({ proposal, onOpen }: ProposalCardProps) => {
  const { timeFormat } = usePreferences().preferences;
  const day = formatDate(new Date(proposal.starts_at));
  const time = (value: string) => formatTime(value, { preference: timeFormat });
  const when = proposal.is_all_day
    ? `${day}, all day`
    : `${day} ${time(proposal.starts_at)}–${time(proposal.ends_at)}`;

  return (
    <div className="assistant-proposal">
      <strong>{proposal.title}</strong>
      <span>{when}</span>
      {proposal.location && <span>{proposal.location}</span>}
      {proposal.warnings?.map((warning) => (
        <span key={warning} className="assistant-warning">
          {warning}
        </span>
      ))}
      <button type="button" className="primary-button" onClick={onOpen}>
        Open in form
      </button>
    </div>
  );
};
