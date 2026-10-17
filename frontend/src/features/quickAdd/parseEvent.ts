import { api } from "../../lib/api";
import type { ParseEventResponse } from "../../types";

// Only reads the text into a draft; nothing is saved until the user submits
// the prefilled event form.
export const parseEvent = (text: string) =>
  api<ParseEventResponse>("/api/events/parse", {
    method: "POST",
    body: JSON.stringify({ text }),
  });
