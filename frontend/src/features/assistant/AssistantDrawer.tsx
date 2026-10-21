import { Drawer } from "../../components/Drawer";
import { cx } from "../../lib/classNames";
import { getErrorText } from "../../lib/requestErrors";
import { ActionCard } from "./ActionCard";
import { MessageForm } from "./MessageForm";
import type { AssistantConversation } from "./useAssistantConversation";

type AssistantDrawerProps = {
  conversation: AssistantConversation;
  onOpenEvent: (eventId: string, startsAt: string) => void;
  onClose: () => void;
};

// A conversation with the calendar assistant. It creates events right away,
// and asks before changing existing ones.
export const AssistantDrawer = ({
  conversation,
  onOpenEvent,
  onClose,
}: AssistantDrawerProps) => {
  const {
    turns,
    ended,
    pendingText,
    sendError,
    send,
    deciding,
    failedDecision,
    decide,
    reset,
  } = conversation;
  const pending = pendingText !== undefined;
  return (
    <Drawer
      eyebrow="Asks before changing your events"
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
                {turn.actions.map((action) => (
                  <ActionCard
                    key={action.id}
                    action={action}
                    busy={deciding}
                    error={
                      failedDecision?.actionId === action.id
                        ? getErrorText(failedDecision.error, "Couldn't do that")
                        : undefined
                    }
                    onDecide={(decision) => decide(action, decision)}
                    onOpen={() =>
                      onOpenEvent(action.event_id, action.starts_at)
                    }
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
      {sendError && (
        <p className="error" role="alert">
          {getErrorText(sendError, "Couldn't reach the assistant")}
        </p>
      )}
      {ended ? (
        <p className="form-hint">This conversation has ended.</p>
      ) : (
        <MessageForm pending={pending} onSend={send} />
      )}
      {turns.length > 0 && !pending && (
        <button type="button" className="text-button" onClick={reset}>
          New conversation
        </button>
      )}
    </Drawer>
  );
};
