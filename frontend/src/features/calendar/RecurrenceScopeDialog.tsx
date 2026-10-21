import type { OccurrenceScope } from "../timeline/timelineRequests";

type ScopeAction = "save" | "move" | "delete";

export type ScopeQuestion = {
  action: ScopeAction;
  // A change to more than the time or skip cannot be made to one occurrence.
  allowThis: boolean;
};

const TITLES: Record<ScopeAction, string> = {
  save: "Save repeating event",
  move: "Move repeating event",
  delete: "Delete repeating event",
};

const SCOPES: [OccurrenceScope, string][] = [
  ["this", "This event"],
  ["following", "This and following events"],
  ["all", "All events"],
];

type RecurrenceScopeDialogProps = ScopeQuestion & {
  onAnswer: (scope: OccurrenceScope | null) => void;
};

// Asks which occurrences of a repeating event a change applies to. Escape,
// Cancel or a press on the backdrop answers null.
export const RecurrenceScopeDialog = ({
  action,
  allowThis,
  onAnswer,
}: RecurrenceScopeDialogProps) => (
  <div
    className="scope-dialog-backdrop"
    onMouseDown={(event) =>
      event.target === event.currentTarget && onAnswer(null)
    }
    onKeyDown={(event) => event.key === "Escape" && onAnswer(null)}
  >
    <div
      className="scope-dialog"
      role="dialog"
      aria-modal="true"
      aria-labelledby="scope-dialog-title"
    >
      <h2 id="scope-dialog-title">{TITLES[action]}</h2>
      {SCOPES.filter(([scope]) => allowThis || scope !== "this").map(
        ([scope, label], index) => (
          <button
            key={scope}
            type="button"
            className={action === "delete" ? "danger-button" : "primary-button"}
            // The narrowest choice is the default.
            autoFocus={index === 0}
            onClick={() => onAnswer(scope)}
          >
            {label}
          </button>
        ),
      )}
      <button
        type="button"
        className="text-button"
        onClick={() => onAnswer(null)}
      >
        Cancel
      </button>
    </div>
  </div>
);
