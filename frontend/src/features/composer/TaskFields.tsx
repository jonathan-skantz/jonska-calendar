import { DateTimeField } from "../../components/DateTimeField";
import { inputDateTime } from "../../lib/calendar";
import type { TimelineEntry } from "../../types";
import { usePreferences } from "../calendar/preferencesContext";
import type { FieldState } from "./fieldState";

type TaskFieldsProps = {
  initial?: Partial<TimelineEntry>;
  // A clicked hour becomes the deadline of a new task.
  draftEventStart: string | null;
  selectedDate: string;
  fields: FieldState;
};

export const TaskFields = ({
  initial,
  draftEventStart,
  selectedDate,
  fields,
}: TaskFieldsProps) => {
  const { timeFormat } = usePreferences().preferences;
  return (
    <label className={fields.labelClass("deadline_at")}>
      Deadline
      <DateTimeField
        name="deadline_at"
        label="Deadline"
        invalid={fields.isInvalid("deadline_at")}
        describedBy={fields.describedBy("deadline_at")}
        defaultValue={
          inputDateTime(initial?.deadline_at) ||
          inputDateTime(draftEventStart ?? undefined) ||
          `${selectedDate}T17:00`
        }
        preference={timeFormat}
        required
      />
    </label>
  );
};
