import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vitest";
import { readRecurrence } from "../../lib/recurrence";
import type { Recurrence } from "../../types";
import { getFieldState } from "./fieldState";
import { RepeatFields } from "./RepeatFields";

afterEach(cleanup);

// Tuesday 13 October 2026: the 2nd Tuesday of its month.
const tuesday = new Date(2026, 9, 13);

const renderFields = (start = tuesday, initial?: Recurrence) => {
  const view = render(
    <form data-testid="form">
      <RepeatFields
        initial={initial}
        start={start}
        fields={getFieldState({}, false)}
      />
    </form>,
  );
  const rule = () =>
    readRecurrence(new FormData(screen.getByTestId("form") as HTMLFormElement));
  return { ...view, rule };
};

describe("RepeatFields", () => {
  it("does not repeat by default", () => {
    const { rule } = renderFields();
    expect(rule()).toBeNull();
    expect(screen.queryByLabelText("Ends")).toBeNull();
  });

  it("repeats weekly on the start's weekday and chosen ones", () => {
    const { rule } = renderFields();
    fireEvent.change(screen.getByLabelText("Repeat"), {
      target: { value: "weekly" },
    });
    const tuesdayChip = screen.getByRole("button", { name: "Tue" });
    expect(tuesdayChip.getAttribute("aria-pressed")).toBe("true");
    expect((tuesdayChip as HTMLButtonElement).disabled).toBe(true);

    fireEvent.click(screen.getByRole("button", { name: "Thu" }));
    fireEvent.change(screen.getByLabelText("Ends"), {
      target: { value: "after" },
    });
    fireEvent.change(screen.getByLabelText("Times"), {
      target: { value: "6" },
    });
    expect(rule()).toEqual({
      frequency: "weekly",
      interval: 1,
      weekdays: ["TU", "TH"],
      count: 6,
    });
  });

  it("offers the monthly days the start allows and falls back when it moves", () => {
    const { rule, rerender } = renderFields();
    fireEvent.change(screen.getByLabelText("Repeat"), {
      target: { value: "monthly" },
    });
    fireEvent.change(screen.getByLabelText("Day of the month"), {
      target: { value: "weekday" },
    });
    expect(rule()).toMatchObject({ monthly_by: "weekday" });

    // The 29th is a 5th Thursday, which has no "nth weekday" option.
    rerender(
      <form data-testid="form">
        <RepeatFields
          start={new Date(2026, 9, 29)}
          fields={getFieldState({}, false)}
        />
      </form>,
    );
    expect(
      Array.from(
        (screen.getByLabelText("Day of the month") as HTMLSelectElement)
          .options,
      ).map((option) => option.text),
    ).toEqual(["On day 29", "On the last Thursday"]);
    expect(rule()).toMatchObject({ monthly_by: "day_of_month" });
  });

  it("shows a saved rule", () => {
    renderFields(tuesday, {
      frequency: "weekly",
      interval: 2,
      weekdays: ["TU", "FR"],
      until: "2026-12-31",
    });
    expect(
      (screen.getByLabelText("Repeat every how many weeks") as HTMLInputElement)
        .value,
    ).toBe("2");
    expect(
      screen.getByRole("button", { name: "Fri" }).getAttribute("aria-pressed"),
    ).toBe("true");
    expect((screen.getByLabelText("Last date") as HTMLInputElement).value).toBe(
      "2026-12-31",
    );
  });
});
