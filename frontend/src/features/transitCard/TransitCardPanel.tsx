import { getErrorText } from "../../lib/requestErrors";
import { useHousehold } from "../household/useHousehold";
import { TransitCardDrawer } from "./TransitCardDrawer";
import { useSaveTransitCardDays, useTransitCardViews } from "./useTransitCard";

type TransitCardPanelProps = {
  dates: Date[];
  today: Date;
  userId: string;
  onClose: () => void;
};

// The transit card drawer with its save. Mounted only while open, so each
// opening starts without the last save's error.
export const TransitCardPanel = ({
  dates,
  today,
  userId,
  onClose,
}: TransitCardPanelProps) => {
  const views = useTransitCardViews(dates, today, userId) ?? [];
  const cardCount = useHousehold().data?.transit_card_count;
  const save = useSaveTransitCardDays();

  return (
    <TransitCardDrawer
      views={views}
      cardCount={cardCount}
      error={getErrorText(save.error, "Unable to save transit card days")}
      pending={save.isPending}
      onClose={onClose}
      onSave={(submission) => save.mutate(submission, { onSuccess: onClose })}
    />
  );
};
