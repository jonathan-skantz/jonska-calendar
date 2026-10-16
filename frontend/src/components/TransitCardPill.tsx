import type { TransitCardPillVariant } from "../features/transitCard/transitCardDays";
import { BusIcon } from "./BusIcon";

type TransitCardPillProps = {
  name: string;
  variant: TransitCardPillVariant;
  column: number;
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
}: TransitCardPillProps) => (
  <span
    className={`transit-card-pill ${variant}`}
    style={{ gridColumn: column + 1 }}
    title={describe(name, variant)}
  >
    <BusIcon crossed={variant === "not_needed"} />
    <span className="all-day-title">
      {variant === "maybe" ? `${name}?` : name}
    </span>
  </span>
);
