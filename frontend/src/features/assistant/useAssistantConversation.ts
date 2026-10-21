import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useState } from "react";
import type { WeekStartsOn } from "../../lib/calendar";
import { invalidateWorkspace } from "../../lib/queryInvalidation";
import type {
  AssistantAction,
  AssistantDecision,
  AssistantReply,
} from "../../types";
import { decideAction, sendAssistantMessage } from "./assistantApi";

export type AssistantTurn =
  | { role: "user"; text: string }
  | ({ role: "assistant" } & Omit<AssistantReply, "conversation_id">);

type Decision = { action: AssistantAction; decision: AssistantDecision };

const replaceAction = (turns: AssistantTurn[], updated: AssistantAction) =>
  turns.map((turn) =>
    turn.role === "assistant"
      ? {
          ...turn,
          actions: turn.actions.map((action) =>
            action.id === updated.id ? updated : action,
          ),
        }
      : turn,
  );

// The conversation lives above the drawer, so it survives closing the drawer
// to open an event. A message joins the thread with its answer; until then it
// is shown from the pending mutation.
export const useAssistantConversation = (weekStartsOn: WeekStartsOn) => {
  const queryClient = useQueryClient();
  const [turns, setTurns] = useState<AssistantTurn[]>([]);
  const [conversationId, setConversationId] = useState<string | null>(null);
  const [ended, setEnded] = useState(false);

  const send = useMutation({
    mutationFn: (text: string) =>
      sendAssistantMessage(text, conversationId, weekStartsOn),
    onSuccess: ({ conversation_id, ...reply }, text) => {
      setConversationId(conversation_id);
      setEnded(reply.ended);
      setTurns((current) => [
        ...current,
        { role: "user", text },
        { role: "assistant", ...reply },
      ]);
      // Created events are already saved.
      if (reply.actions.some((action) => action.status === "done"))
        void invalidateWorkspace(queryClient);
    },
  });

  // Actions only exist after a reply, which set the conversation id.
  const decide = useMutation({
    mutationFn: ({ action, decision }: Decision) =>
      decideAction(conversationId!, action.id, decision),
    onSuccess: (updated, { decision }) => {
      setTurns((current) => replaceAction(current, updated));
      if (decision !== "decline") void invalidateWorkspace(queryClient);
    },
  });

  const reset = () => {
    setTurns([]);
    setConversationId(null);
    setEnded(false);
    send.reset();
    decide.reset();
  };

  return {
    turns,
    ended,
    pendingText: send.isPending ? send.variables : undefined,
    sendError: send.error,
    send: send.mutateAsync,
    deciding: decide.isPending,
    // A failed decision belongs to its action's card, until the next one.
    failedDecision: decide.isError
      ? { actionId: decide.variables.action.id, error: decide.error }
      : undefined,
    decide: (action: AssistantAction, decision: AssistantDecision) =>
      decide.mutate({ action, decision }),
    reset,
  };
};

export type AssistantConversation = ReturnType<typeof useAssistantConversation>;
