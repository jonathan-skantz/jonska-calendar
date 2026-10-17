import { getRequestErrorMessage } from "./requestErrors";

export const api = async <T>(
  path: string,
  options?: RequestInit,
): Promise<T> => {
  const response = await fetch(path, {
    credentials: "include",
    headers: { "Content-Type": "application/json", ...options?.headers },
    ...options,
  });
  if (!response.ok)
    throw new Error((await response.text()) || "Request failed");
  if (response.status === 204) return undefined as T;
  return response.json() as Promise<T>;
};

type SendForErrorOptions = {
  method: string;
  body?: unknown;
  fallbackMessage: string;
  // Runs after a successful request, typically query invalidation.
  onSuccess: () => Promise<unknown>;
};

// For actions whose callers show errors inline: resolves to an error message,
// or null when the request and onSuccess both succeeded.
export const sendForError = async (
  path: string,
  { method, body, fallbackMessage, onSuccess }: SendForErrorOptions,
): Promise<string | null> => {
  try {
    await api(path, {
      method,
      body: body === undefined ? undefined : JSON.stringify(body),
    });
    await onSuccess();
    return null;
  } catch (requestError) {
    return getRequestErrorMessage(requestError, fallbackMessage);
  }
};
