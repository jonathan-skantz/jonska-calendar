import { useQueryClient } from "@tanstack/react-query";
import { useActionMutation } from "../../hooks/useActionMutation";
import { invalidateWorkspace } from "../../lib/queryInvalidation";
import type { Customization, TimelineEntry } from "../../types";
import { customizationPath } from "./customization";

// Each action resolves to an error message, or null when it succeeded. The
// customized event is in the timeline and its reminders, so both reload.
export const useCustomizationMutations = (entry: TimelineEntry) => {
  const queryClient = useQueryClient();
  const run = useActionMutation("Unable to save your changes", () =>
    invalidateWorkspace(queryClient),
  );
  const path = customizationPath(entry);
  return {
    save: (customization: Customization) =>
      run({ path, method: "PUT", body: customization }),
    reset: () => run({ path, method: "DELETE" }),
  };
};
