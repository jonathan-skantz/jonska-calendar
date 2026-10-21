import { useState } from "react";
import type { ReactNode } from "react";

type MoreDetailsProps = {
  // Open from the start, e.g. when an edited item already uses a field here.
  defaultOpen: boolean;
  // A field inside failed validation; the section opens so it can be seen.
  hasError: boolean;
  children: ReactNode;
};

// Optional fields folded away to keep the form short. A closed <details>
// keeps its controls in the form, so their values are still submitted.
export const MoreDetails = ({
  defaultOpen,
  hasError,
  children,
}: MoreDetailsProps) => {
  const [open, setOpen] = useState(defaultOpen);
  if (hasError && !open) setOpen(true);
  return (
    <details
      className="more-details"
      open={open}
      onToggle={(event) => setOpen(event.currentTarget.open)}
    >
      <summary>More details</summary>
      <div className="more-details-fields">{children}</div>
    </details>
  );
};
