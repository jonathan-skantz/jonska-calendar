export const getRequestErrorMessage = (
  error: unknown,
  fallback: string,
): string => (error instanceof Error ? error.message : fallback);
