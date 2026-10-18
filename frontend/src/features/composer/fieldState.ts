import type { ComposerField, ComposerFieldErrors } from "./formValidation";

export const COMPOSER_ERROR_ID = "composer-error";

// How each field shows a validation error: its label is highlighted, and its
// control is marked invalid and points at the form's error message.
export const getFieldState = (
  fieldErrors: ComposerFieldErrors,
  hasErrorMessage: boolean,
) => {
  const isInvalid = (field: ComposerField) => Boolean(fieldErrors[field]);
  const describedBy = (field: ComposerField) =>
    isInvalid(field) && hasErrorMessage ? COMPOSER_ERROR_ID : undefined;
  return {
    isInvalid,
    describedBy,
    errorId: hasErrorMessage ? COMPOSER_ERROR_ID : undefined,
    labelClass: (field: ComposerField) =>
      isInvalid(field) ? "field-invalid" : undefined,
    inputProps: (field: ComposerField) => ({
      "aria-invalid": isInvalid(field),
      "aria-describedby": describedBy(field),
    }),
  };
};

export type FieldState = ReturnType<typeof getFieldState>;
