import { getTransitCardPillVariant } from "./transitCardDays";
import type {
  TransitCardDayView,
  TransitCardPillVariant,
} from "./transitCardDays";
import { BusIcon } from "./BusIcon";
import { cx } from "../../lib/classNames";

type TransitCardPillProps = {
  name: string;
  variant: TransitCardPillVariant;
  // Grid column in the week's Card row; omitted for inline pills.
  column?: number;
  compact?: boolean;
  // Only the name's initial, for the narrow week columns.
  short?: boolean;
};

const describe = (name: string, variant: TransitCardPillVariant) => {
  switch (variant) {
    case "pending":
      return `${name} has not reported this day yet`;
    case "not_needed":
      return `${name} doesn't need the transit card`;
    case "maybe":
      return `${name} might need the transit card`;
    case "conflict":
      return `${name} needs the transit card (conflict)`;
    case "resolved":
      return `${name} needs the transit card`;
  }
};

export const TransitCardPill = ({
  name,
  variant,
  column,
  compact = false,
  short = false,
}: TransitCardPillProps) => (
  <span
    className={cx(
      "transit-card-pill",
      variant,
      compact && "compact",
      short && "short",
    )}
    style={column === undefined ? undefined : { gridColumn: column + 1 }}
    title={describe(name, variant)}
  >
    <BusIcon crossed={variant === "not_needed"} />
    <span className="all-day-title">
      {short ? name.slice(0, 1).toUpperCase() : name}
      {variant === "maybe" && "?"}
    </span>
  </span>
);

// The week header's Card row: a pill in each day's column that has an answer
// or is still waiting for one.
export const TransitCardDayPills = ({
  views,
  fallbackName,
}: {
  views: TransitCardDayView[];
  // Names the current user on days they have not reported yet.
  fallbackName: string;
}) =>
  views.map((view, index) => {
    const variant = getTransitCardPillVariant(view);
    return variant ? (
      <TransitCardPill
        key={view.key}
        name={view.day?.display_name ?? fallbackName}
        variant={variant}
        column={index}
        short
      />
    ) : null;
  });
