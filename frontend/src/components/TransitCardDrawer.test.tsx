import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { TransitCardDrawer } from "./TransitCardDrawer";
import { getTransitCardDayViews } from "../features/transitCard/transitCardDays";

const dates = [19, 20, 21, 22, 23, 24, 25].map((day) => new Date(2026, 9, day));

const renderDrawer = (onSave = vi.fn()) => {
  const views = getTransitCardDayViews(dates, [], new Date(2026, 9, 21, 9));
  render(
    <TransitCardDrawer
      views={views}
      error=""
      onClose={vi.fn()}
      onSave={onSave}
    />,
  );
  return onSave;
};

describe("TransitCardDrawer", () => {
  afterEach(cleanup);

  it("submits an empty selection as an explicit report", () => {
    const onSave = renderDrawer();

    fireEvent.click(
      screen.getByRole("button", { name: "Save – no days needed" }),
    );

    expect(onSave).toHaveBeenCalledWith({
      from: "2026-10-21",
      to: "2026-10-25",
      needed_dates: [],
      maybe_dates: [],
    });
  });

  it("disables days that have ended and submits yes and maybe answers", () => {
    const onSave = renderDrawer();

    const yesButtons = screen.getAllByRole<HTMLButtonElement>("button", {
      name: "Yes",
    });
    expect(yesButtons.map((button) => button.disabled)).toEqual([
      true,
      true,
      false,
      false,
      false,
      false,
      false,
    ]);
    fireEvent.click(yesButtons[3]);
    fireEvent.click(screen.getAllByRole("button", { name: "Maybe" })[4]);
    fireEvent.click(screen.getByRole("button", { name: "Save" }));

    expect(onSave).toHaveBeenCalledWith({
      from: "2026-10-21",
      to: "2026-10-25",
      needed_dates: ["2026-10-22"],
      maybe_dates: ["2026-10-23"],
    });
  });
});
