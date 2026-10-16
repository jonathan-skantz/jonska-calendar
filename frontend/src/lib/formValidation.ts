type ComposerType = "event" | "task";

export type ComposerField =
  | "title"
  | "description"
  | "url"
  | "kind"
  | "starts_at"
  | "ends_at"
  | "location"
  | "commute_before_minutes"
  | "commute_after_minutes"
  | "deadline_at"
  | "event-tags"
  | "main-tag";

export type ComposerFieldErrors = Partial<Record<ComposerField, true>>;

const isComposerField = (name: string): name is ComposerField =>
  [
    "title",
    "description",
    "url",
    "kind",
    "starts_at",
    "ends_at",
    "location",
    "commute_before_minutes",
    "commute_after_minutes",
    "deadline_at",
    "event-tags",
    "main-tag",
  ].includes(name);

export const getComposerFieldErrors = (
  form: HTMLFormElement,
  composerType: ComposerType,
  selectedTagIds: string[],
  mainTagId: string | null,
) => {
  const invalidNames = Array.from(form.elements)
    .filter(
      (
        control,
      ): control is
        HTMLInputElement | HTMLSelectElement | HTMLTextAreaElement =>
        control instanceof HTMLInputElement ||
        control instanceof HTMLSelectElement ||
        control instanceof HTMLTextAreaElement,
    )
    .filter((control) => !control.checkValidity())
    .map((control) => control.dataset.field || control.name)
    .filter(Boolean);

  const startsValue = String(
    (form.elements.namedItem("starts_at") as HTMLInputElement | null)?.value ??
      "",
  );
  const endsValue = String(
    (form.elements.namedItem("ends_at") as HTMLInputElement | null)?.value ??
      "",
  );
  const isAllDay =
    (form.elements.namedItem("is_all_day") as HTMLInputElement | null)
      ?.value === "true";
  const invalidEventRange = isAllDay
    ? endsValue.slice(0, 10) < startsValue.slice(0, 10)
    : new Date(endsValue) <= new Date(startsValue);
  if (
    composerType === "event" &&
    startsValue &&
    endsValue &&
    invalidEventRange
  ) {
    invalidNames.push("starts_at", "ends_at");
  }
  if (composerType === "event" && selectedTagIds.length === 0)
    invalidNames.push("event-tags");
  if (
    composerType === "event" &&
    (!mainTagId || !selectedTagIds.includes(mainTagId))
  ) {
    invalidNames.push("main-tag");
  }

  const errors: ComposerFieldErrors = {};
  for (const name of new Set(invalidNames)) {
    if (isComposerField(name)) errors[name] = true;
  }
  return errors;
};
