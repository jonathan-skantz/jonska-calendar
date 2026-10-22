import type { Customization, TimelineEntry } from "../../types";

export const TEXT_FIELDS = [
  { name: "title", label: "Title" },
  { name: "location", label: "Location" },
  { name: "url", label: "URL" },
  { name: "description", label: "Details", multiline: true },
  { name: "note", label: "My note", multiline: true },
] as const;

type TextField = (typeof TEXT_FIELDS)[number]["name"];

// The form keeps numbers as text, in whole minutes. Shifts may be negative
// (earlier); commute and gathering may not. A blank gathering keeps the
// received one and 0 removes it.
export type CustomizationDraft = Record<TextField, string> & {
  startShiftMinutes: string;
  endShiftMinutes: string;
  commuteBeforeMinutes: string;
  commuteAfterMinutes: string;
  gatheringMinutes: string;
  gatheringIsOptional: boolean;
  gatheringLabel: string;
};

export type MinutesField = {
  [Key in keyof CustomizationDraft]: Key extends `${string}Minutes`
    ? Key
    : never;
}[keyof CustomizationDraft];

const minutes = (seconds?: number) => (seconds ? String(seconds / 60) : "");

export const draftFrom = (
  customization: Customization = {},
): CustomizationDraft => ({
  ...(Object.fromEntries(
    TEXT_FIELDS.map(({ name }) => [name, customization[name] ?? ""]),
  ) as Record<TextField, string>),
  startShiftMinutes: minutes(customization.start_shift_seconds),
  endShiftMinutes: minutes(customization.end_shift_seconds),
  commuteBeforeMinutes: minutes(customization.commute_before_seconds),
  commuteAfterMinutes: minutes(customization.commute_after_seconds),
  gatheringMinutes:
    customization.gathering_seconds === undefined
      ? ""
      : String(customization.gathering_seconds / 60),
  gatheringIsOptional: Boolean(customization.gathering_is_optional),
  gatheringLabel: customization.gathering_label ?? "",
});

// Blank text and zero or blank minutes keep the received value. Minutes that
// are not whole, or negative where they cannot be, are null, for the form
// to refuse.
export const customizationFrom = (
  draft: CustomizationDraft,
): Customization | null => {
  const seconds = (value: string, canBeNegative = false) => {
    const parsed = Number(value.trim() || 0);
    return Number.isInteger(parsed) && (canBeNegative || parsed >= 0)
      ? parsed * 60
      : null;
  };
  const start = seconds(draft.startShiftMinutes, true);
  const end = seconds(draft.endShiftMinutes, true);
  const commuteBefore = seconds(draft.commuteBeforeMinutes);
  const commuteAfter = seconds(draft.commuteAfterMinutes);
  const gathering = draft.gatheringMinutes.trim()
    ? seconds(draft.gatheringMinutes)
    : undefined;
  if (
    start === null ||
    end === null ||
    commuteBefore === null ||
    commuteAfter === null ||
    gathering === null
  )
    return null;
  const customization: Customization = {};
  for (const { name } of TEXT_FIELDS) {
    const value = draft[name].trim();
    if (value) customization[name] = value;
  }
  if (start) customization.start_shift_seconds = start;
  if (end) customization.end_shift_seconds = end;
  if (commuteBefore) customization.commute_before_seconds = commuteBefore;
  if (commuteAfter) customization.commute_after_seconds = commuteAfter;
  if (gathering !== undefined) {
    customization.gathering_seconds = gathering;
    const label = draft.gatheringLabel.trim();
    if (gathering && draft.gatheringIsOptional)
      customization.gathering_is_optional = true;
    if (gathering && label) customization.gathering_label = label;
  }
  return customization;
};

export const customizationPath = (entry: TimelineEntry) =>
  entry.subscription
    ? `/api/subscribed-events/${entry.id}/customization`
    : `/api/events/${entry.id}/customization`;
