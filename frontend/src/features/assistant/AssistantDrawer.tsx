import { Drawer } from "../../components/Drawer";
import { cx } from "../../lib/classNames";
import type { AssistantProposal } from "../../types";
import { MessageForm } from "./MessageForm";
import { ProposalCard } from "./ProposalCard";
import type { AssistantTurn } from "./useAssistantConversation";

type AssistantDrawerProps = {
  turns: AssistantTurn[];
  ended: boolean;
  // The message waiting for an answer.
  pendingText?: string;
  error: string;
  onSend: (text: string) => Promise<unknown>;
  onReset: () => void;
  onOpenProposal: (proposal: AssistantProposal) => void;
  onClose: () => void;
};

// A conversation with the calendar assistant. It reads the calendar and
// proposes events; each proposal opens in the event form for the user to
// review and save.
export const AssistantDrawer = ({
  turns,
  ended,
  pendingText,
  error,
  onSend,
  onReset,
  onOpenProposal,
  onClose,
}: AssistantDrawerProps) => {
  const pending = pendingText !== undefined;
  return (
    <Drawer
      eyebrow="Reads your calendar, saves nothing"
      title="Assistant"
      className="assistant-drawer"
      onClose={onClose}
    >
      <ol className="assistant-thread" aria-live="polite">
        {turns.length === 0 && !pending && (
          <li className="form-hint">
            Ask about your week or plan something, e.g. "Lunch with Sam on
            Friday, or Monday if I'm busy".
          </li>
        )}
        {turns.map((turn, index) => (
          <li
            key={index}
            className={cx("assistant-turn", turn.role === "user" && "user")}
          >
            {turn.role === "user" ? (
              turn.text
            ) : (
              <>
                {turn.steps.length > 0 && (
                  <ul className="assistant-steps" aria-label="What it checked">
                    {turn.steps.map((step, stepIndex) => (
                      <li key={stepIndex}>{step.summary}</li>
                    ))}
                  </ul>
                )}
                {turn.reply && <p>{turn.reply}</p>}
                {turn.proposals.map((proposal, proposalIndex) => (
                  <ProposalCard
                    key={proposalIndex}
                    proposal={proposal}
                    onOpen={() => onOpenProposal(proposal)}
                  />
                ))}
              </>
            )}
          </li>
        ))}
        {pending && (
          <>
            <li className="assistant-turn user">{pendingText}</li>
            <li className="assistant-status">Thinking…</li>
          </>
        )}
      </ol>
      {error && (
        <p className="error" role="alert">
          {error}
        </p>
      )}
      {ended ? (
        <p className="form-hint">This conversation has ended.</p>
      ) : (
        <MessageForm pending={pending} onSend={onSend} />
      )}
      {turns.length > 0 && !pending && (
        <button type="button" className="text-button" onClick={onReset}>
          New conversation
        </button>
      )}
    </Drawer>
  );
};
