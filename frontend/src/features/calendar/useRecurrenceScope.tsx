import { useState } from "react";
import type { OccurrenceScope } from "../timeline/timelineRequests";
import { RecurrenceScopeDialog } from "./RecurrenceScopeDialog";
import type { ScopeQuestion } from "./RecurrenceScopeDialog";

type PendingQuestion = ScopeQuestion & {
  resolve: (scope: OccurrenceScope | null) => void;
};

// ask resolves to the chosen scope, or null when the user cancels; dialog is
// the element to render while a question is open.
export const useRecurrenceScope = () => {
  const [pending, setPending] = useState<PendingQuestion | null>(null);
  const ask = (question: ScopeQuestion) =>
    new Promise<OccurrenceScope | null>((resolve) =>
      setPending({ ...question, resolve }),
    );
  const dialog = pending && (
    <RecurrenceScopeDialog
      action={pending.action}
      allowThis={pending.allowThis}
      onAnswer={(scope) => {
        pending.resolve(scope);
        setPending(null);
      }}
    />
  );
  return { ask, dialog };
};
