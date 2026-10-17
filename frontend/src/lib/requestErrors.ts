export const getRequestErrorMessage = (
  error: unknown,
  fallback: string,
): string => (error instanceof Error ? error.message : fallback);

// The message to show for a failed mutation, or "" when it has not failed.
export const getErrorText = (error: unknown, fallback: string) =>
  error ? getRequestErrorMessage(error, fallback) : "";
