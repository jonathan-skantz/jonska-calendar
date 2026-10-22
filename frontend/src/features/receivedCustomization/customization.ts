import type { Customization, TimelineEntry } from "../../types";

export const TEXT_FIELDS = [
  { name: "title", label: "Title" },
  { name: "location", label: "Location" },
  { name: "url", label: "URL" },
  { name: "description", label: "Details", multiline: true },
  { name: "note", label: "My note", multiline: true },
] as const;

type TextField = (typeof TEXT_FIELDS)[number]["name"];

// The form keeps every field as text: shifts are whole minutes, negative is
// earlier.
export type CustomizationDraft = Record<TextField, string> & {
  startShiftMinutes: string;
  endShiftMinutes: string;
};

const minutes = (seconds?: number) => (seconds ? String(seconds / 60) : "");

export const draftFrom = (
  customization: Customization = {},
): CustomizationDraft => ({
  ...(Object.fromEntries(
    TEXT_FIELDS.map(({ name }) => [name, customization[name] ?? ""]),
  ) as Record<TextField, string>),
  startShiftMinutes: minutes(customization.start_shift_seconds),
  endShiftMinutes: minutes(customization.end_shift_seconds),
});

// Blank text and zero or blank shifts keep the received value. A shift that
// is not a whole number of minutes is null, for the form to refuse.
export const customizationFrom = (
  draft: CustomizationDraft,
): Customization | null => {
  const shift = (value: string) => {
    const parsed = Number(value.trim() || 0);
    return Number.isInteger(parsed) ? parsed * 60 : null;
  };
  const start = shift(draft.startShiftMinutes);
  const end = shift(draft.endShiftMinutes);
  if (start === null || end === null) return null;
  const customization: Customization = {};
  for (const { name } of TEXT_FIELDS) {
    const value = draft[name].trim();
    if (value) customization[name] = value;
  }
  if (start) customization.start_shift_seconds = start;
  if (end) customization.end_shift_seconds = end;
  return customization;
};

export const customizationPath = (entry: TimelineEntry) =>
  entry.subscription
    ? `/api/subscribed-events/${entry.id}/customization`
    : `/api/events/${entry.id}/customization`;
