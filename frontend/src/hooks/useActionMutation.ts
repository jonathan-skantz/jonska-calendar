import { useMutation } from "@tanstack/react-query";
import { api } from "../lib/api";
import { getRequestErrorMessage } from "../lib/requestErrors";

type ActionRequest = { path: string; method: string; body?: unknown };

// One mutation for a group of related endpoints whose panel shows each
// action's outcome inline. Each call resolves to an error message, or null
// when the request and onSuccess both succeeded.
export const useActionMutation = (
  fallbackMessage: string,
  onSuccess: () => Promise<unknown>,
) => {
  const { mutateAsync } = useMutation({
    mutationFn: ({ path, method, body }: ActionRequest) =>
      api(path, {
        method,
        body: body === undefined ? undefined : JSON.stringify(body),
      }),
    onSuccess,
  });

  return async (request: ActionRequest): Promise<string | null> => {
    try {
      await mutateAsync(request);
      return null;
    } catch (requestError) {
      return getRequestErrorMessage(requestError, fallbackMessage);
    }
  };
};
