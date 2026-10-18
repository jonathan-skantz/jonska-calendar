import { api } from "../../lib/api";
import type { WeekStartsOn } from "../../lib/calendar";
import type { ParseEventResponse } from "../../types";
import { resolveDraft } from "./resolveDraft";

// Only reads the text into a draft; nothing is saved until the user submits
// the prefilled event form.
const parseEvent = (text: string) =>
  api<ParseEventResponse>("/api/events/parse", {
    method: "POST",
    body: JSON.stringify({ text }),
  });

// Reads quick-add text into composer values, with notes on everything the
// server or the browser could not place.
export const readEventDraft = async (
  text: string,
  weekStartsOn: WeekStartsOn,
  selectedDate: string,
) => {
  const result = await parseEvent(text);
  const resolved = resolveDraft(
    result.draft,
    new Date(),
    weekStartsOn,
    selectedDate,
  );
  return {
    ...resolved,
    warnings: [
      ...result.unresolved.map((part) => `Couldn't place "${part}"`),
      ...result.warnings,
      ...resolved.warnings,
    ],
  };
};
