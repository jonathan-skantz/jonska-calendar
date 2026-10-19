import { api } from "../../lib/api";
import { SECONDS_PER_MINUTE, type WeekStartsOn } from "../../lib/calendar";
import type { AssistantProposal, AssistantReply } from "../../types";
import type { ResolvedDraft } from "../quickAdd/resolveDraft";

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

// A proposal opens in the event form the same way a quick add draft does.
export const proposalToDraft = (
  proposal: AssistantProposal,
): ResolvedDraft => ({
  values: {
    title: proposal.title,
    starts_at: proposal.starts_at,
    ends_at: proposal.ends_at,
    is_all_day: proposal.is_all_day,
    location: proposal.location,
    kind: proposal.kind,
  },
  tagIds: proposal.tag_ids ?? [],
  reminderOffsets: (proposal.reminder_minutes ?? []).map(
    (minutes) => minutes * SECONDS_PER_MINUTE,
  ),
  warnings: proposal.warnings ?? [],
});
