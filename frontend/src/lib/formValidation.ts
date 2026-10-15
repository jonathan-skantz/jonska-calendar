type ComposerType = "event" | "task";

export const getInvalidComposerFields = (
  form: HTMLFormElement,
  composerType: ComposerType,
  selectedTagIds: string[],
  mainTagId: string | null,
) => {
  const invalidNames = Array.from(form.elements)
    .filter(
      (control): control is HTMLInputElement | HTMLSelectElement | HTMLTextAreaElement =>
        control instanceof HTMLInputElement ||
        control instanceof HTMLSelectElement ||
        control instanceof HTMLTextAreaElement,
    )
    .filter((control) => !control.checkValidity())
    .map((control) => control.name)
    .filter(Boolean);

  const startsValue = String(
    (form.elements.namedItem("starts_at") as HTMLInputElement | null)?.value ?? "",
  );
  const endsValue = String(
    (form.elements.namedItem("ends_at") as HTMLInputElement | null)?.value ?? "",
  );
  if (
    composerType === "event" &&
    startsValue &&
    endsValue &&
    new Date(endsValue) <= new Date(startsValue)
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

  return [...new Set(invalidNames)];
};
