import { api } from "../../lib/api";
import type { WeekStartsOn } from "../../lib/calendar";
import type {
  AssistantAction,
  AssistantDecision,
  AssistantReply,
} from "../../types";

// The server keeps the conversation; the browser only sends its id with the
// next message. The time zone and week start are the browser's, because the
// server cannot know them.
export const sendAssistantMessage = (
  text: string,
  conversationId: string | null,
  weekStartsOn: WeekStartsOn,
) =>
  api<AssistantReply>("/api/assistant/messages", {
    method: "POST",
    body: JSON.stringify({
      conversation_id: conversationId ?? undefined,
      text,
      time_zone: Intl.DateTimeFormat().resolvedOptions().timeZone,
      week_start: weekStartsOn,
    }),
  });

// Applies or declines a pending change, or undoes a created event. Returns
// the action with its new status.
export const decideAction = (
  conversationId: string,
  actionId: string,
  decision: AssistantDecision,
) =>
  api<AssistantAction>(
    `/api/assistant/conversations/${conversationId}/actions/${actionId}`,
    { method: "POST", body: JSON.stringify({ decision }) },
  );
