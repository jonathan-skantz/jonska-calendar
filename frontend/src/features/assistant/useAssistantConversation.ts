import { useMutation } from "@tanstack/react-query";
import { useState } from "react";
import type { WeekStartsOn } from "../../lib/calendar";
import type { AssistantReply } from "../../types";
import { sendAssistantMessage } from "./assistantApi";

export type AssistantTurn =
  | { role: "user"; text: string }
  | ({ role: "assistant" } & Omit<AssistantReply, "conversation_id">);

// The conversation lives above the drawer, so it survives closing the drawer
// to open a proposal in the event form. A message joins the thread with its
// answer; until then it is shown from the pending mutation.
export const useAssistantConversation = (weekStartsOn: WeekStartsOn) => {
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
    },
  });

  const reset = () => {
    setTurns([]);
    setConversationId(null);
    setEnded(false);
    send.reset();
  };

  return {
    turns,
    ended,
    pendingText: send.isPending ? send.variables : undefined,
    error: send.error,
    send: send.mutateAsync,
    reset,
  };
};
